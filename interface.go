///////////////////////////////////////////////
///////////////////////////////////////////////
// SHIVAM KAK
// 31 MARCH 2025
// DUST TASK
///////////////////////////////////////////////
// FILE: interface.go

// Interface.go converts the typescript
// interface of the throttler provided in the
// TASK.md to equivalent structs in go.
// Additionally, it defines all types relevant
// to the request-throttler implementation.
///////////////////////////////////////////////
package throttler

import (
    "sync"
    "time"
)

type RateLimit struct {
	RequestCount  int // Number of requests allowed per rolling window
	WindowSeconds int // Time window defined in seconds
}

type Connection struct {
	Platform   string    // A unique platform identifier
	Connection string    // A unique connection identifier
	Niceness   int       // Associated workflow priority, 0 is the highest priority
	RateLimit  RateLimit // The rate-limit for the connection
}

// RequestItem represents a request with priority information
type requestItem struct {
	function  func()        // The function to execute
	niceness  int           // Priority (lower is higher priority)
	timestamp time.Time     // Timestamp for FIFO ordering within same priority
	doneCh    chan struct{} // Channel to signal completion
	index     int           // Used by heap package
}

// priorityQueue implements heap.Interface for priority-based request scheduling
type priorityQueue []*requestItem

// ConnectionRateLimiter manages rate limiting for a specific connection
type connectionRateLimiter struct {
	rateLimit   RateLimit
	tokens      float64   // Available tokens (using float for fractional refill)
	lastRefill  time.Time // Last time tokens were refilled
	queue       priorityQueue
	requestCh   chan *requestItem
	quit        chan struct{}
	mu          sync.Mutex
}

// Registry manages rate limiters for all connections
// maintains a map indexed by conn.Platform + ":" + conn.Connection
type rateLimiterRegistry struct {
	limiters map[string]*connectionRateLimiter
	mu       sync.Mutex
}

