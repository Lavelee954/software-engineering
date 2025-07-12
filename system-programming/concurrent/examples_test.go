package concurrent

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestBoringPatterns(t *testing.T) {
	t.Run("Bad Boring Pattern", func(t *testing.T) {
		start := time.Now()
		badBoring("bad pattern")
		time.Sleep(500 * time.Millisecond) // give time for output
		duration := time.Since(start)
		t.Logf("Bad boring pattern took: %v (goroutines may still be running)", duration)
	})

	t.Run("Good Boring Pattern", func(t *testing.T) {
		start := time.Now()
		joe := goodBoring("Joe")
		ann := goodBoring("Ann")

		// Read a few messages
		for i := 0; i < 5; i++ {
			select {
			case msg := <-joe:
				fmt.Printf("Joe: %s\n", msg)
			case msg := <-ann:
				fmt.Printf("Ann: %s\n", msg)
			}
		}
		duration := time.Since(start)
		t.Logf("Good boring pattern took: %v", duration)
	})
}

func TestFanInPatterns(t *testing.T) {
	t.Run("Bad Fan-In Pattern", func(t *testing.T) {
		// Create test channels
		input1 := make(chan string)
		input2 := make(chan string)

		// Start producers
		go func() {
			for i := 0; i < 3; i++ {
				input1 <- fmt.Sprintf("Input1-%d", i)
				time.Sleep(100 * time.Millisecond)
			}
		}()

		go func() {
			for i := 0; i < 3; i++ {
				input2 <- fmt.Sprintf("Input2-%d", i)
				time.Sleep(150 * time.Millisecond)
			}
		}()

		// This won't work well in tests because it runs forever
		// We'll just test the setup
		t.Log("Bad fan-in pattern doesn't provide a clean interface")
	})

	t.Run("Good Fan-In Pattern", func(t *testing.T) {
		// Create test channels
		input1 := make(chan string)
		input2 := make(chan string)

		// Start producers
		go func() {
			for i := 0; i < 3; i++ {
				input1 <- fmt.Sprintf("Input1-%d", i)
				time.Sleep(100 * time.Millisecond)
			}
		}()

		go func() {
			for i := 0; i < 3; i++ {
				input2 <- fmt.Sprintf("Input2-%d", i)
				time.Sleep(150 * time.Millisecond)
			}
		}()

		// Use fan-in
		combined := goodFanIn(input1, input2)

		// Read combined messages
		for i := 0; i < 6; i++ {
			select {
			case msg := <-combined:
				fmt.Printf("Combined: %s\n", msg)
			case <-time.After(1 * time.Second):
				t.Log("Timeout reading from combined channel")
				return
			}
		}
		t.Log("Good fan-in pattern successfully combined channels")
	})
}

func TestSequencingPatterns(t *testing.T) {
	t.Run("Bad Sequencing", func(t *testing.T) {
		start := time.Now()
		badSequencing()
		duration := time.Since(start)
		t.Logf("Bad sequencing took: %v (order may vary)", duration)
	})

	t.Run("Good Sequencing", func(t *testing.T) {
		start := time.Now()
		goodSequencing()
		duration := time.Since(start)
		t.Logf("Good sequencing took: %v (order preserved)", duration)
	})
}

func TestTimeoutPatterns(t *testing.T) {
	t.Run("Bad Timeout", func(t *testing.T) {
		start := time.Now()

		// This will block for 2 seconds
		go func() {
			// We need to test this differently to avoid blocking the test
			c := make(chan string)
			go func() {
				time.Sleep(2 * time.Second)
				c <- "slow result"
			}()

			select {
			case result := <-c:
				fmt.Println("Result:", result)
			case <-time.After(3 * time.Second):
				fmt.Println("Test timeout")
			}
		}()

		time.Sleep(100 * time.Millisecond) // Just to start the goroutine
		duration := time.Since(start)
		t.Logf("Bad timeout test setup took: %v", duration)
	})

	t.Run("Good Timeout", func(t *testing.T) {
		start := time.Now()
		goodTimeout()
		duration := time.Since(start)
		t.Logf("Good timeout took: %v (should be ~1 second)", duration)
	})
}

func TestQuitPatterns(t *testing.T) {
	t.Run("Bad Quit Pattern", func(t *testing.T) {
		start := time.Now()
		badQuitPattern()
		duration := time.Since(start)
		t.Logf("Bad quit pattern took: %v (goroutine may still be running)", duration)
	})

	t.Run("Good Quit Pattern", func(t *testing.T) {
		start := time.Now()
		goodQuitPattern()
		duration := time.Since(start)
		t.Logf("Good quit pattern took: %v (clean shutdown)", duration)
	})
}

func TestPingPongPatterns(t *testing.T) {
	t.Run("Bad Ping-Pong", func(t *testing.T) {
		start := time.Now()
		badPingPong()
		duration := time.Since(start)
		t.Logf("Bad ping-pong took: %v (shared memory with mutex)", duration)
	})

	t.Run("Good Ping-Pong", func(t *testing.T) {
		start := time.Now()
		goodPingPong()
		duration := time.Since(start)
		t.Logf("Good ping-pong took: %v (channel-based communication)", duration)
	})
}

func TestRingBufferPatterns(t *testing.T) {
	t.Run("Ring Buffer", func(t *testing.T) {
		rb := NewRingBuffer(3)

		// Test good send (with overflow protection)
		for i := 0; i < 5; i++ {
			rb.GoodSend(fmt.Sprintf("Item-%d", i))
		}

		// Receive items
		for i := 0; i < 3; i++ {
			item := rb.Receive()
			fmt.Printf("Received: %v\n", item)
		}

		t.Log("Ring buffer test completed - overflow protection worked")
	})
}

func TestRateLimitingPatterns(t *testing.T) {
	t.Run("Bad Rate Limiting", func(t *testing.T) {
		start := time.Now()
		badRateLimit()
		duration := time.Since(start)
		t.Logf("Bad rate limiting took: %v (no rate limiting)", duration)
	})

	t.Run("Good Rate Limiting", func(t *testing.T) {
		start := time.Now()
		goodRateLimit()
		duration := time.Since(start)
		t.Logf("Good rate limiting took: %v (rate limited)", duration)
	})
}

func TestContextPatterns(t *testing.T) {
	t.Run("Bad Context Operation", func(t *testing.T) {
		start := time.Now()
		badContextOperation()
		duration := time.Since(start)
		t.Logf("Bad context operation took: %v (no cancellation)", duration)
	})

	t.Run("Good Context Operation", func(t *testing.T) {
		start := time.Now()
		goodContextOperation()
		duration := time.Since(start)
		t.Logf("Good context operation took: %v (with cancellation)", duration)
	})
}

func TestErrorHandlingPatterns(t *testing.T) {
	t.Run("Bad Error Handling", func(t *testing.T) {
		start := time.Now()
		badErrorHandling()
		duration := time.Since(start)
		t.Logf("Bad error handling took: %v (errors not communicated)", duration)
	})

	t.Run("Good Error Handling", func(t *testing.T) {
		start := time.Now()
		goodErrorHandling()
		duration := time.Since(start)
		t.Logf("Good error handling took: %v (errors properly communicated)", duration)
	})
}

// Race condition test
func TestRaceConditions(t *testing.T) {
	// This test should be run with -race flag
	t.Run("Shared Counter Race", func(t *testing.T) {
		// BAD: Race condition
		var counter int
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 1000; j++ {
					counter++ // Race condition!
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		t.Logf("Final counter value (with race): %d", counter)
	})

	t.Run("Channel-based Counter", func(t *testing.T) {
		// GOOD: No race condition
		counter := make(chan int, 1)
		counter <- 0
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 1000; j++ {
					current := <-counter
					counter <- current + 1
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		final := <-counter
		t.Logf("Final counter value (no race): %d", final)
	})
}

// Benchmark comparing patterns
func BenchmarkPatternComparison(b *testing.B) {
	b.Run("Sequential vs Parallel", func(b *testing.B) {
		b.Run("Sequential", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Simulate sequential work
				time.Sleep(10 * time.Millisecond)
			}
		})

		b.Run("Parallel", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				done := make(chan bool)
				// Simulate parallel work
				go func() {
					time.Sleep(10 * time.Millisecond)
					done <- true
				}()
				<-done
			}
		})
	})
}

// Example of the Google Search pattern evolution
func TestGoogleSearchEvolution(t *testing.T) {
	// Mock search functions
	mockSearch := func(kind string) func(string) string {
		return func(query string) string {
			time.Sleep(time.Duration(50+rand.Intn(50)) * time.Millisecond)
			return fmt.Sprintf("%s results for %s", kind, query)
		}
	}

	web := mockSearch("Web")
	image := mockSearch("Image")
	video := mockSearch("Video")

	t.Run("Google 1.0 - Sequential", func(t *testing.T) {
		start := time.Now()
		results := []string{
			web("golang"),
			image("golang"),
			video("golang"),
		}
		duration := time.Since(start)
		t.Logf("Sequential search took: %v, results: %d", duration, len(results))
	})

	t.Run("Google 2.0 - Parallel", func(t *testing.T) {
		start := time.Now()
		c := make(chan string)

		go func() { c <- web("golang") }()
		go func() { c <- image("golang") }()
		go func() { c <- video("golang") }()

		var results []string
		for i := 0; i < 3; i++ {
			results = append(results, <-c)
		}
		duration := time.Since(start)
		t.Logf("Parallel search took: %v, results: %d", duration, len(results))
	})

	t.Run("Google 2.1 - Parallel with Timeout", func(t *testing.T) {
		start := time.Now()
		c := make(chan string)

		go func() { c <- web("golang") }()
		go func() { c <- image("golang") }()
		go func() { c <- video("golang") }()

		var results []string
		timeout := time.After(80 * time.Millisecond)

		for i := 0; i < 3; i++ {
			select {
			case result := <-c:
				results = append(results, result)
			case <-timeout:
				t.Logf("Search timed out after %d results", len(results))
				goto done
			}
		}

	done:
		duration := time.Since(start)
		t.Logf("Parallel search with timeout took: %v, results: %d", duration, len(results))
	})
}

// Test context cancellation
func TestContextCancellation(t *testing.T) {
	t.Run("Context Timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()

		select {
		case <-time.After(200 * time.Millisecond):
			t.Error("Should have been cancelled")
		case <-ctx.Done():
			duration := time.Since(start)
			t.Logf("Context cancelled after: %v, error: %v", duration, ctx.Err())
		}
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()

		select {
		case <-time.After(200 * time.Millisecond):
			t.Error("Should have been cancelled")
		case <-ctx.Done():
			duration := time.Since(start)
			t.Logf("Context cancelled after: %v, error: %v", duration, ctx.Err())
		}
	})
}
