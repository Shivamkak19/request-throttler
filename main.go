///////////////////////////////////////////////
///////////////////////////////////////////////
// SHIVAM KAK
// 31 MARCH 2025
// DUST TASK
///////////////////////////////////////////////
// FILE: main.go

// Main.go handles the core implementation of
// the request-throttler. 
// Throttle calls findOrCreateLimiter 
// to get a limiter from globalRegistry,
// which calls newConnectionRateLimiter 
// if needed. The limiter's submit method
// adds requests to a priority queue. 
// A worker goroutine runs processRequests 
// continuously, calling refillTokens 
// and executing queued functions when tokens
// are available, maintaining 
// rate limits and priority ordering.

// IT IS RECOMMENDED TO READ THE DESIGN.TXT
// BEFORE PROCEEDING TO READ THIS FILE.
///////////////////////////////////////////////
package throttler

import (
	"container/heap"
	"time"
)

// Create a new rate limiter registry
func newRateLimiterRegistry() *rateLimiterRegistry {
	return &rateLimiterRegistry{
		limiters: make(map[string]*connectionRateLimiter),
	}
}

// Global registry instance
// Accessed in core Throttle function
var globalRegistry = newRateLimiterRegistry()

// Creates a new connection rate limiter
func newConnectionRateLimiter(rateLimit RateLimit) *connectionRateLimiter {
	rl := &connectionRateLimiter{
		rateLimit:   rateLimit,
		
		// Start with full tokens
		tokens:      float64(rateLimit.RequestCount), 
		lastRefill:  time.Now(),
		queue:       make(priorityQueue, 0),
		
		// Buffer size can be adjusted
		requestCh:   make(chan *requestItem, 100), 
		quit:        make(chan struct{}),
	}
	
	// Initialize priority queue
	heap.Init(&rl.queue)
	
	// Start the worker goroutine
	go rl.processRequests()
	
	return rl
}

// Refills the token bucket based on elapsed time
func (rl *connectionRateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	
	if elapsed > 0 {
		// Calculate tokens to add
		refillRate := float64(rl.rateLimit.RequestCount) / float64(rl.rateLimit.WindowSeconds)
		tokensToAdd := elapsed * refillRate
		
		rl.tokens += tokensToAdd
		// Cap at maximum tokens
		if rl.tokens > float64(rl.rateLimit.RequestCount) {
			rl.tokens = float64(rl.rateLimit.RequestCount)
		}
		rl.lastRefill = now
	}
}

// Main worker routine that processes requests based on priority and rate limits
func (rl *connectionRateLimiter) processRequests() {
	// Check frequency: arbitrarily set to 10 ms
	ticker := time.NewTicker(10 * time.Millisecond) 
	defer ticker.Stop()
	
	for {
		select {
		case req := <-rl.requestCh:
			// Add new request to priority queue
			rl.mu.Lock()
			heap.Push(&rl.queue, req)
			rl.mu.Unlock()
			
		case <-ticker.C:
			rl.mu.Lock()
			// Refill tokens
			rl.refillTokens()
			
			// Process as many requests as we have tokens for
			for rl.tokens >= 1.0 && rl.queue.Len() > 0 {
				// Get highest priority request
				req := heap.Pop(&rl.queue).(*requestItem)
				rl.tokens--
				
				// Execute in a separate goroutine
				go func(r *requestItem) {
					r.function()
					close(r.doneCh) // Signal completion
				}(req)
			}
			rl.mu.Unlock()
			
		case <-rl.quit:
			return
		}
	}
}

// Submit a request to be processed based on priority
func (rl *connectionRateLimiter) submit(fn func(), niceness int) chan struct{} {
	doneCh := make(chan struct{})
	
	req := &requestItem{
		function:  fn,
		niceness:  niceness,
		timestamp: time.Now(),
		doneCh:    doneCh,
	}
	
	rl.requestCh <- req
	
	return doneCh
}

// Stop the rate limiter
func (rl *connectionRateLimiter) stop() {
	close(rl.quit)
}

// Get or create a rate limiter for a connection
func (r *rateLimiterRegistry) findOrCreateLimiter(conn Connection) *connectionRateLimiter {
	key := conn.Platform + ":" + conn.Connection
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	limiter, exists := r.limiters[key]
	if !exists {
		limiter = newConnectionRateLimiter(conn.RateLimit)
		r.limiters[key] = limiter
	}
	
	return limiter
}

// Throttle throttles a function call based on the connection's rate limit and priority
func Throttle[T any, U any](connection Connection, fn func(arg T) (U, error)) func(arg T) (U, error) {
	return func(arg T) (U, error) {
		// Get the limiter for this connection
		limiter := globalRegistry.findOrCreateLimiter(connection)
		
		// Variables to store the result
		var result U
		var err error
		
		// Create the function to execute
		execFn := func() {
			result, err = fn(arg)
		}
		
		// Submit the request and wait for it to complete
		doneCh := limiter.submit(execFn, connection.Niceness)
		<-doneCh
		
		return result, err
	}
}