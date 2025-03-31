///////////////////////////////////////////////
///////////////////////////////////////////////
// SHIVAM KAK
// 31 MARCH 2025
// DUST TASK
///////////////////////////////////////////////
// FILE: client.go

// Client.go provides a very simple client
// that demonstrates usage of the
// request-throttler.

// To use, run the following:
// go build
// go run client/client.go
///////////////////////////////////////////////
package main

import (
    "fmt"
    "throttler"
)

func main() {
    // VERY SIMPLE Example usage of request-throttler
    conn := throttler.Connection{
        Platform:   "example",
        Connection: "conn1",
        Niceness:   1,
        RateLimit: throttler.RateLimit{
            RequestCount:  10,
            WindowSeconds: 60,
        },
    }
    
    // usage of the Throttle interface function defined in TASK.md
    throttledFunc := throttler.Throttle(conn, func(s string) (string, error) {
        return "Processed: " + s, nil
    })
    
    result, err := throttledFunc("test")
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Result:", result)
    }
}