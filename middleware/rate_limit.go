package middleware

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
)

// visitor tracks a single key's fixed-window count.
type visitor struct {
	count       int
	windowStart time.Time
}

// Limiter is a fixed-window limiter with a static max.
// The limit comes from internal/ratelimit (no DB dependency).
type Limiter struct {
	max        int
	window     time.Duration
	visitorsMu sync.Mutex
	visitors   map[string]*visitor
}

// NewLimiter creates a limiter allowing max requests per ratelimit.Window.
// Falls back to ratelimit.DefaultPerMinute when max < 1.
func NewLimiter(max int) *Limiter {
	if max < 1 {
		max = ratelimit.DefaultPerMinute
	}
	return &Limiter{
		max:      max,
		window:   ratelimit.Window,
		visitors: make(map[string]*visitor),
	}
}

// rateLimitKey prefers the authenticated user (set by RequireAuth) over the
// client IP. Keying on the JWT subject means a caller cannot multiply their
// quota by rotating X-Forwarded-For. IP is a fallback only for requests that
// reach the limiter without a user (e.g. if middleware order changes).
func rateLimitKey(c *fiber.Ctx) string {
	if userID, ok := c.Locals("userID").(string); ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.IP()
}

// Handler is a fixed-window limiter keyed by user (preferred) or IP.
// Register it after RequireAuth on authenticated routes.
func (d *Limiter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := rateLimitKey(c)
		now := time.Now()

		d.visitorsMu.Lock()
		v, ok := d.visitors[key]
		if !ok || now.Sub(v.windowStart) >= d.window {
			d.visitors[key] = &visitor{count: 1, windowStart: now}
			size := len(d.visitors)
			d.visitorsMu.Unlock()
			// Opportunistic cleanup to bound memory.
			if size%1000 == 0 {
				go d.cleanup()
			}
			return c.Next()
		}
		v.count++
		count := v.count
		retryAfter := int(math.Ceil(v.windowStart.Add(d.window).Sub(now).Seconds()))
		d.visitorsMu.Unlock()

		if count > d.max {
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set(fiber.HeaderRetryAfter, fmt.Sprintf("%d", retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Rate limit exceeded: %d requests per minute allowed", d.max),
			})
		}
		return c.Next()
	}
}

func (d *Limiter) cleanup() {
	cutoff := time.Now().Add(-2 * d.window)
	d.visitorsMu.Lock()
	defer d.visitorsMu.Unlock()
	for key, v := range d.visitors {
		if v.windowStart.Before(cutoff) {
			delete(d.visitors, key)
		}
	}
}
