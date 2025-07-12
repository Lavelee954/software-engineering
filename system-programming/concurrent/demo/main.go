package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== Go Concurrency Pattern Comparison Demo ===")
	fmt.Println()

	// Example 1: Goroutine synchronization
	fmt.Println("1. Goroutine Synchronization:")
	fmt.Println("   BAD:  Goroutines without synchronization - may not complete")
	fmt.Println("   GOOD: Channel-based synchronization - ensures completion")
	fmt.Println()

	// Example 2: Search patterns
	fmt.Println("2. Search Service Performance:")
	fmt.Println("   Sequential: ~200ms (sum of all operations)")
	fmt.Println("   Parallel:   ~80ms  (maximum of concurrent operations)")
	fmt.Println("   With timeout: Responsive even if some operations are slow")
	fmt.Println()

	// Example 3: Worker pools
	fmt.Println("3. Worker Pool Pattern:")
	fmt.Println("   BAD:  Create goroutine per job - high resource usage")
	fmt.Println("   GOOD: Fixed number of workers - bounded resource usage")
	fmt.Println()

	// Example 4: Context
	fmt.Println("4. Context Usage:")
	fmt.Println("   BAD:  No cancellation - operations continue indefinitely")
	fmt.Println("   GOOD: Context cancellation - operations can be stopped")
	fmt.Println()

	// Example 5: Race conditions
	fmt.Println("5. Race Condition Prevention:")
	fmt.Println("   BAD:  Shared memory without protection - race conditions")
	fmt.Println("   GOOD: Channel-based communication - no race conditions")
	fmt.Println()

	// Example 6: Channel patterns
	fmt.Println("6. Channel Design:")
	fmt.Println("   Generator: Functions return channels (receive-only)")
	fmt.Println("   Fan-in:    Combine multiple channels into one")
	fmt.Println("   Timeout:   Use select with time.After for timeouts")
	fmt.Println("   Quit:      Graceful shutdown with quit channels")
	fmt.Println()

	// Example 7: Common mistakes
	fmt.Println("7. Common Mistakes to Avoid:")
	fmt.Println("   • Goroutine leaks (no termination condition)")
	fmt.Println("   • Race conditions (unprotected shared state)")
	fmt.Println("   • Deadlocks (channel read/write imbalance)")
	fmt.Println("   • No cancellation (ignoring context)")
	fmt.Println("   • Unbounded parallelism (too many goroutines)")
	fmt.Println()

	// Example 8: Best practices
	fmt.Println("8. Best Practices:")
	fmt.Println("   • Don't communicate by sharing memory; share memory by communicating")
	fmt.Println("   • Use channels for synchronization and communication")
	fmt.Println("   • Always plan goroutine lifecycle")
	fmt.Println("   • Use context for cancellation and deadlines")
	fmt.Println("   • Implement graceful shutdown")
	fmt.Println("   • Use race detector: go test -race")
	fmt.Println()

	// Demonstrate a simple good pattern
	fmt.Println("9. Live Demo - Good Pattern:")
	done := make(chan bool)

	go func() {
		for i := 0; i < 3; i++ {
			fmt.Printf("   Worker processing item %d\n", i+1)
			time.Sleep(100 * time.Millisecond)
		}
		done <- true
	}()

	fmt.Println("   Waiting for worker to complete...")
	<-done
	fmt.Println("   Worker completed successfully!")
	fmt.Println()

	fmt.Println("=== Run 'go test -v' to see all pattern comparisons ===")
	fmt.Println("=== Run 'go test -race -v' to detect race conditions ===")
	fmt.Println("=== Run 'go test -bench .' to see performance benchmarks ===")
}
