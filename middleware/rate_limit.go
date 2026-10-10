package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
)

// visitor tracks a single IP's fixed-window count.
type visitor struct {
	count       int
	windowStart time.Time
}

// Limiter is a fixed-window per-IP limiter with a static max.
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

// Handler is a fixed-window per-IP limiter.
func (d *Limiter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		max := d.max
		ip := c.IP()
		now := time.Now()

		d.visitorsMu.Lock()
		v, ok := d.visitors[ip]
		if !ok || now.Sub(v.windowStart) >= d.window {
			d.visitors[ip] = &visitor{count: 1, windowStart: now}
			d.visitorsMu.Unlock()
			// Opportunistic cleanup to bound memory.
			if len(d.visitors)%1000 == 0 {
				go d.cleanup()
			}
			return c.Next()
		}
		v.count++
		count := v.count
		d.visitorsMu.Unlock()

		if count > max {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Rate limit exceeded: %d requests per minute allowed", max),
			})
		}
		return c.Next()
	}
}

func (d *Limiter) cleanup() {
	cutoff := time.Now().Add(-2 * time.Minute)
	d.visitorsMu.Lock()
	defer d.visitorsMu.Unlock()
	for ip, v := range d.visitors {
		if v.windowStart.Before(cutoff) {
			delete(d.visitors, ip)
		}
	}
}
