///////////////////////////////////////////////
///////////////////////////////////////////////
// SHIVAM KAK
// 31 MARCH 2025
// DUST TASK
///////////////////////////////////////////////
// FILE: unit_test.go

// Unit_test.go provides unit tests for 
// the request-throttler. The tests check
// functionality of rate limiting
// and priority ordering.

// TESTS:
// 1. TestBasicThrottle
// 2. TestRateLimiting
// 3. TestPriorityOrdering
// 4. TestMultipleConnections

// run all tests with:
// go build
// cd client
// go test
///////////////////////////////////////////////
package main

import (
	"fmt"
	"sync"
	"testing"
	"throttler"
	"time"
)

// TestBasicThrottle tests that the throttler executes functions correctly
func TestBasicThrottle(t *testing.T) {
	conn := throttler.Connection{
		Platform:   "test",
		Connection: "basic",
		Niceness:   1,
		RateLimit: throttler.RateLimit{
			RequestCount:  10,
			WindowSeconds: 1,
		},
	}

	// Create a throttled function
	throttledFunc := throttler.Throttle(conn, func(s string) (string, error) {
		return "Processed: " + s, nil
	})

	// Execute the throttled function
	result, err := throttledFunc("test-input")
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "Processed: test-input" {
		t.Fatalf("Expected 'Processed: test-input', got: %s", result)
	}
}

// TestRateLimiting tests that the throttler correctly rate limits requests
func TestRateLimiting(t *testing.T) {
	conn := throttler.Connection{
		Platform:   "test",
		Connection: "rate-limit",
		Niceness:   1,
		RateLimit: throttler.RateLimit{
			RequestCount:  5,      // Only 5 requests allowed
			WindowSeconds: 1,      // Per 1 second window
		},
	}

	// Create a throttled function with a sleep to make timing more reliable
	throttledFunc := throttler.Throttle(conn, func(i int) (int, error) {
		time.Sleep(10 * time.Millisecond) // Small sleep to ensure consistent timing
		return i * 2, nil
	})

	// Create variables to track execution times
	start := time.Now()
	var firstBatchEnd time.Time
	var secondBatchEnd time.Time

	// Execute first batch - should all be quick
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			result, _ := throttledFunc(val)
			if result != val*2 {
				t.Errorf("Expected %d, got: %d", val*2, result)
			}
		}(i)
	}
	wg.Wait()
	firstBatchEnd = time.Now()

	// Execute second batch - should be throttled
	for i := 5; i < 10; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			result, _ := throttledFunc(val)
			if result != val*2 {
				t.Errorf("Expected %d, got: %d", val*2, result)
			}
		}(i)
	}
	wg.Wait()
	secondBatchEnd = time.Now()

	// Verify timing
	firstBatchDuration := firstBatchEnd.Sub(start)
	secondBatchDuration := secondBatchEnd.Sub(firstBatchEnd)

	fmt.Printf("First batch duration: %v\n", firstBatchDuration)
	fmt.Printf("Second batch duration: %v\n", secondBatchDuration)

	// The second batch should take longer as it waits for rate limiting
	if secondBatchDuration <= firstBatchDuration {
		t.Errorf("Expected second batch to take longer due to rate limiting")
	}
}

// TestPriorityOrdering tests that requests 
// with higher priority (lower niceness) are processed first
func TestPriorityOrdering(t *testing.T) {
    // Create two connections with different priorities but using the same connection key
    // This ensures they use the same rate limiter instance
    highPriorityConn := throttler.Connection{
        Platform:   "test",
        Connection: "priority-test", // Same connection for both
        Niceness:   0,              // Higher priority (lower niceness)
        RateLimit: throttler.RateLimit{
            RequestCount:  1,        // Only 1 request allowed per second
            WindowSeconds: 1,
        },
    }

    lowPriorityConn := throttler.Connection{
        Platform:   "test",
        Connection: "priority-test", // Same connection for both
        Niceness:   10,             // Lower priority (higher niceness)
        RateLimit: throttler.RateLimit{
            RequestCount:  1,        // Only 1 request allowed per second
            WindowSeconds: 1,
        },
    }

    // Sequence to track order of execution
    var sequence []string
    var mu sync.Mutex

    // Create throttled functions with longer work time to make test more reliable
    highPriorityFunc := throttler.Throttle(highPriorityConn, func(s string) (string, error) {
        mu.Lock()
        sequence = append(sequence, "high")
        mu.Unlock()
        time.Sleep(100 * time.Millisecond) // Longer sleep
        return "high-" + s, nil
    })

    lowPriorityFunc := throttler.Throttle(lowPriorityConn, func(s string) (string, error) {
        mu.Lock()
        sequence = append(sequence, "low")
        mu.Unlock()
        time.Sleep(100 * time.Millisecond) // Longer sleep
        return "low-" + s, nil
    })

    // Submit multiple low priority tasks first
    var wg sync.WaitGroup
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            lowPriorityFunc("task")
        }()
    }

    // Wait to ensure low priority tasks are queued
    time.Sleep(50 * time.Millisecond)

    // Submit high priority task
    wg.Add(1)
    go func() {
        defer wg.Done()
        highPriorityFunc("task")
    }()

    // Wait for all tasks to complete
    wg.Wait()

    // Check execution sequence
    mu.Lock()
    defer mu.Unlock()

    fmt.Printf("Execution sequence: %v\n", sequence)
    
    if len(sequence) < 4 {
        t.Fatalf("Expected at least 4 tasks to be executed, got %d", len(sequence))
    }
    
    // Find position of high priority task
    highPos := -1
    for i, item := range sequence {
        if item == "high" {
            highPos = i
            break
        }
    }
    
    if highPos == -1 {
        t.Fatalf("High priority task was not executed")
    }
    
    // There should be at least one low priority task after the high priority one
    lowAfterHigh := false
    for i := highPos + 1; i < len(sequence); i++ {
        if sequence[i] == "low" {
            lowAfterHigh = true
            break
        }
    }
    
    // Also verify there's at least one low priority task before high (to prove ordering)
    lowBeforeHigh := false
    for i := 0; i < highPos; i++ {
        if sequence[i] == "low" {
            lowBeforeHigh = true
            break
        }
    }
    
    if !lowBeforeHigh {
        t.Errorf("Expected at least one low priority task to execute before high priority")
    }
    
    if !lowAfterHigh {
        t.Errorf("Expected at least one low priority task to execute after high priority")
    }
}

// TestMultipleConnections tests that different connections are rate limited independently
func TestMultipleConnections(t *testing.T) {
	conn1 := throttler.Connection{
		Platform:   "test",
		Connection: "conn1",
		Niceness:   1,
		RateLimit: throttler.RateLimit{
			RequestCount:  3,
			WindowSeconds: 1,
		},
	}

	conn2 := throttler.Connection{
		Platform:   "test",
		Connection: "conn2",
		Niceness:   1,
		RateLimit: throttler.RateLimit{
			RequestCount:  3, 
			WindowSeconds: 1,
		},
	}

	var executed1, executed2 int
	var mu sync.Mutex

	// Create throttled functions
	throttledFunc1 := throttler.Throttle(conn1, func(i int) (int, error) {
		mu.Lock()
		executed1++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return i, nil
	})

	throttledFunc2 := throttler.Throttle(conn2, func(i int) (int, error) {
		mu.Lock()
		executed2++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return i, nil
	})

	// Execute functions concurrently
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(2)
		go func(val int) {
			defer wg.Done()
			throttledFunc1(val)
		}(i)
		go func(val int) {
			defer wg.Done()
			throttledFunc2(val)
		}(i)
	}

	// Wait a short time to let some execute (not all should complete due to rate limiting)
	time.Sleep(100 * time.Millisecond)

	// Check that both connections have executed tasks, demonstrating
	// that they're being processed independently
	mu.Lock()
	defer mu.Unlock()
	
	if executed1 == 0 || executed2 == 0 {
		t.Errorf("Expected both connections to execute tasks, but got conn1: %d, conn2: %d", 
			executed1, executed2)
	}
}