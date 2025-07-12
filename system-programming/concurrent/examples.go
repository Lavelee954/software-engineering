package concurrent

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Example 1: Basic Boring Generator Pattern (Pattern 3 from guide)
// BAD: Simple goroutine without channel
func badBoring(msg string) {
	go func() {
		for i := 0; ; i++ {
			fmt.Printf("%s %d\n", msg, i)
			time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
		}
	}()
}

// GOOD: Generator pattern with channel
func goodBoring(msg string) <-chan string {
	c := make(chan string)
	go func() {
		for i := 0; ; i++ {
			c <- fmt.Sprintf("%s %d", msg, i)
			time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
		}
	}()
	return c
}

// Example 2: Fan-In Pattern (Pattern 4 from guide)
// BAD: No fan-in, reading from multiple channels separately
func badFanIn(input1, input2 <-chan string) {
	for {
		select {
		case msg1 := <-input1:
			fmt.Println("Input1:", msg1)
		case msg2 := <-input2:
			fmt.Println("Input2:", msg2)
		}
	}
}

// GOOD: Fan-in pattern combining multiple channels
func goodFanIn(input1, input2 <-chan string) <-chan string {
	c := make(chan string)
	go func() {
		for {
			c <- <-input1
		}
	}()
	go func() {
		for {
			c <- <-input2
		}
	}()
	return c
}

// BETTER: Variadic fan-in
func betterFanIn(inputs ...<-chan string) <-chan string {
	c := make(chan string)
	for _, input := range inputs {
		input := input // capture loop variable
		go func() {
			for {
				c <- <-input
			}
		}()
	}
	return c
}

// Example 3: Sequencing Pattern (Pattern 5 from guide)
type Message struct {
	str  string
	wait chan bool
}

// BAD: No sequencing control
func badSequencing() {
	c := make(chan string)
	go func() {
		for i := 0; i < 5; i++ {
			c <- fmt.Sprintf("Message %d", i)
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		}
		close(c)
	}()

	for msg := range c {
		fmt.Println("Received:", msg)
	}
}

// GOOD: Sequencing with wait channels
func goodSequencing() {
	c := make(chan Message)
	go func() {
		for i := 0; i < 5; i++ {
			msg := Message{
				str:  fmt.Sprintf("Message %d", i),
				wait: make(chan bool),
			}
			c <- msg
			<-msg.wait
		}
		close(c)
	}()

	for msg := range c {
		fmt.Println("Processing:", msg.str)
		time.Sleep(100 * time.Millisecond) // simulate work
		msg.wait <- true
	}
}

// Example 4: Timeout Pattern (Pattern 6 from guide)
// BAD: No timeout handling
func badTimeout() {
	c := make(chan string)
	go func() {
		time.Sleep(2 * time.Second) // simulate slow operation
		c <- "slow result"
	}()

	result := <-c // this will block for 2 seconds
	fmt.Println("Result:", result)
}

// GOOD: Timeout with select
func goodTimeout() {
	c := make(chan string)
	go func() {
		time.Sleep(2 * time.Second) // simulate slow operation
		c <- "slow result"
	}()

	select {
	case result := <-c:
		fmt.Println("Result:", result)
	case <-time.After(1 * time.Second):
		fmt.Println("Timeout: operation took too long")
	}
}

// Example 5: Quit Channel Pattern (Pattern 7 from guide)
// BAD: No graceful shutdown
func badQuitPattern() {
	c := make(chan string)
	go func() {
		for i := 0; ; i++ {
			c <- fmt.Sprintf("Message %d", i)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	for i := 0; i < 5; i++ {
		fmt.Println(<-c)
	}
	// goroutine continues running after function exits
}

// GOOD: Quit channel for graceful shutdown
func goodQuitPattern() {
	quit := make(chan bool)
	c := make(chan string)

	go func() {
		for i := 0; ; i++ {
			select {
			case c <- fmt.Sprintf("Message %d", i):
				time.Sleep(100 * time.Millisecond)
			case <-quit:
				fmt.Println("Goroutine shutting down")
				return
			}
		}
	}()

	for i := 0; i < 5; i++ {
		fmt.Println(<-c)
	}
	quit <- true
	time.Sleep(100 * time.Millisecond) // allow goroutine to finish
}

// Example 6: Ping-Pong Pattern (Pattern 13 from guide)
type Ball struct {
	hits int
}

// BAD: Shared memory approach
func badPingPong() {
	ball := &Ball{hits: 0}
	var mu sync.Mutex

	player1 := func() {
		for i := 0; i < 5; i++ {
			mu.Lock()
			ball.hits++
			fmt.Printf("Player1: %d hits\n", ball.hits)
			mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}

	player2 := func() {
		for i := 0; i < 5; i++ {
			mu.Lock()
			ball.hits++
			fmt.Printf("Player2: %d hits\n", ball.hits)
			mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}

	go player1()
	go player2()
	time.Sleep(1 * time.Second)
}

// GOOD: Channel-based ping-pong
func goodPingPong() {
	table := make(chan *Ball)

	player := func(name string) {
		for {
			ball := <-table
			ball.hits++
			fmt.Printf("%s: %d hits\n", name, ball.hits)
			time.Sleep(100 * time.Millisecond)
			table <- ball
		}
	}

	go player("Player1")
	go player("Player2")

	table <- &Ball{hits: 0}
	time.Sleep(1 * time.Second)
}

// Example 7: Ring Buffer Pattern (Pattern 17 from guide)
type RingBuffer struct {
	buffer chan interface{}
	size   int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make(chan interface{}, size),
		size:   size,
	}
}

// BAD: No overflow protection
func (rb *RingBuffer) BadSend(item interface{}) {
	rb.buffer <- item // this blocks if buffer is full
}

// GOOD: Ring buffer with overflow protection
func (rb *RingBuffer) GoodSend(item interface{}) {
	select {
	case rb.buffer <- item:
		// Successfully sent
	default:
		// Buffer full, drop oldest item
		select {
		case <-rb.buffer:
			rb.buffer <- item
		default:
		}
	}
}

func (rb *RingBuffer) Receive() interface{} {
	return <-rb.buffer
}

// Example 8: Rate Limiting Pattern
// BAD: No rate limiting
func badRateLimit() {
	requests := make(chan int)

	go func() {
		for i := 0; i < 10; i++ {
			requests <- i
		}
		close(requests)
	}()

	for req := range requests {
		fmt.Printf("Processing request %d\n", req)
		// Process immediately - no rate limiting
	}
}

// GOOD: Rate limiting with ticker
func goodRateLimit() {
	requests := make(chan int)
	limiter := time.NewTicker(100 * time.Millisecond)
	defer limiter.Stop()

	go func() {
		for i := 0; i < 10; i++ {
			requests <- i
		}
		close(requests)
	}()

	for req := range requests {
		<-limiter.C // wait for rate limiter
		fmt.Printf("Processing request %d\n", req)
	}
}

// Example 9: Context Cancellation Pattern (Pattern 16 from guide)
// BAD: No cancellation support
func badContextOperation() {
	results := make(chan string)

	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(500 * time.Millisecond)
			results <- fmt.Sprintf("Result %d", i)
		}
		close(results)
	}()

	for result := range results {
		fmt.Println("Got:", result)
	}
}

// GOOD: Context with cancellation
func goodContextOperation() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	results := make(chan string)

	go func() {
		defer close(results)
		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("Operation cancelled:", ctx.Err())
				return
			case results <- fmt.Sprintf("Result %d", i):
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	for result := range results {
		fmt.Println("Got:", result)
	}
}

// Example 10: Error Handling Pattern
// BAD: No error handling
func badErrorHandling() {
	results := make(chan string)

	go func() {
		defer close(results)
		for i := 0; i < 5; i++ {
			if i == 3 {
				// Simulate error - but no way to communicate it
				return
			}
			results <- fmt.Sprintf("Result %d", i)
		}
	}()

	for result := range results {
		fmt.Println("Got:", result)
	}
}

// GOOD: Error handling with separate error channel
func goodErrorHandling() {
	results := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(results)
		defer close(errors)
		for i := 0; i < 5; i++ {
			if i == 3 {
				errors <- fmt.Errorf("error at step %d", i)
				return
			}
			results <- fmt.Sprintf("Result %d", i)
		}
	}()

	for {
		select {
		case result, ok := <-results:
			if !ok {
				results = nil
			} else {
				fmt.Println("Got:", result)
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
			} else {
				fmt.Println("Error:", err)
			}
		}
		if results == nil && errors == nil {
			break
		}
	}
}
