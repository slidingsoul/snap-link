// Package ratelimit holds the fixed rate-limit configuration.
// Previously this came from the lookups table (LINK_PER_MINUTE);
// now a hard constant so there is no DB/config dependency.
package ratelimit

import "time"

// DefaultPerMinute is the fixed per-IP limit for POST /snap.
const DefaultPerMinute = 5

// Window is the fixed counting window.
const Window = time.Minute
