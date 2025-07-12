package concurrent

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Example 1: Basic Goroutine Management
// BAD: Goroutine without synchronization
func badGoroutineExample() {
	fmt.Println("Starting bad goroutine example...")
	for i := 0; i < 3; i++ {
		go func(id int) {
			fmt.Printf("Bad goroutine %d: working...\n", id)
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Bad goroutine %d: done\n", id)
		}(i)
	}
	// main returns immediately, goroutines may not complete
	fmt.Println("Bad example finished")
}

// GOOD: Channel-based synchronization
func goodGoroutineExample() {
	fmt.Println("Starting good goroutine example...")
	done := make(chan bool)
	for i := 0; i < 3; i++ {
		go func(id int) {
			fmt.Printf("Good goroutine %d: working...\n", id)
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("Good goroutine %d: done\n", id)
			done <- true
		}(i)
	}
	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
	fmt.Println("Good example finished")
}

// Example 2: Search Service Implementation
type SearchResult struct {
	Source string
	Query  string
	Data   string
	Time   time.Duration
}

func mockWebSearch(query string) SearchResult {
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return SearchResult{
		Source: "web",
		Query:  query,
		Data:   fmt.Sprintf("Web results for %s", query),
		Time:   time.Duration(rand.Intn(100)) * time.Millisecond,
	}
}

func mockImageSearch(query string) SearchResult {
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return SearchResult{
		Source: "image",
		Query:  query,
		Data:   fmt.Sprintf("Image results for %s", query),
		Time:   time.Duration(rand.Intn(100)) * time.Millisecond,
	}
}

func mockVideoSearch(query string) SearchResult {
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return SearchResult{
		Source: "video",
		Query:  query,
		Data:   fmt.Sprintf("Video results for %s", query),
		Time:   time.Duration(rand.Intn(100)) * time.Millisecond,
	}
}

// BAD: Sequential search (slow)
func badSequentialSearch(query string) []SearchResult {
	var results []SearchResult
	results = append(results, mockWebSearch(query))
	results = append(results, mockImageSearch(query))
	results = append(results, mockVideoSearch(query))
	return results
}

// GOOD: Parallel search
func goodParallelSearch(query string) []SearchResult {
	c := make(chan SearchResult)
	go func() { c <- mockWebSearch(query) }()
	go func() { c <- mockImageSearch(query) }()
	go func() { c <- mockVideoSearch(query) }()

	var results []SearchResult
	for i := 0; i < 3; i++ {
		results = append(results, <-c)
	}
	return results
}

// BETTER: Parallel search with timeout
func betterParallelSearchWithTimeout(query string) []SearchResult {
	c := make(chan SearchResult)
	go func() { c <- mockWebSearch(query) }()
	go func() { c <- mockImageSearch(query) }()
	go func() { c <- mockVideoSearch(query) }()

	var results []SearchResult
	timeout := time.After(150 * time.Millisecond)

	for i := 0; i < 3; i++ {
		select {
		case result := <-c:
			results = append(results, result)
		case <-timeout:
			fmt.Printf("Search timed out after %d results\n", len(results))
			return results
		}
	}
	return results
}

// Example 3: Worker Pool Pattern
type Job struct {
	ID   int
	Data string
}

type JobResult struct {
	JobID  int
	Result string
	Error  error
}

func processJob(job Job) JobResult {
	// Simulate work
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return JobResult{
		JobID:  job.ID,
		Result: fmt.Sprintf("Processed: %s", job.Data),
		Error:  nil,
	}
}

// BAD: Create goroutine for each job (resource intensive)
func badJobProcessing(jobs []Job) []JobResult {
	var results []JobResult
	var wg sync.WaitGroup
	resultChan := make(chan JobResult, len(jobs))

	// This creates as many goroutines as jobs - can overwhelm system
	for _, job := range jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			result := processJob(j)
			resultChan <- result
		}(job)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// GOOD: Worker pool with bounded parallelism
func goodWorkerPoolProcessing(jobs []Job) []JobResult {
	const numWorkers = 3
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan JobResult, len(jobs))

	// Start fixed number of workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				result := processJob(job)
				resultChan <- result
			}
		}()
	}

	// Send jobs
	go func() {
		for _, job := range jobs {
			jobChan <- job
		}
		close(jobChan)
	}()

	// Close result channel when all workers done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []JobResult
	for result := range resultChan {
		results = append(results, result)
	}
	return results
}

// Example 4: Context Usage
// BAD: No cancellation support
func badLongRunningOperation(data []string) []string {
	var results []string
	for _, item := range data {
		// Simulate long operation
		time.Sleep(50 * time.Millisecond)
		results = append(results, fmt.Sprintf("processed: %s", item))
	}
	return results
}

// GOOD: Context with cancellation
func goodContextAwareOperation(ctx context.Context, data []string) ([]string, error) {
	var results []string
	for _, item := range data {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			// Simulate work
			time.Sleep(50 * time.Millisecond)
			results = append(results, fmt.Sprintf("processed: %s", item))
		}
	}
	return results, nil
}

// Example 5: Channel Buffering
// BAD: Unbuffered channel causing potential deadlock
func badChannelUsage() {
	ch := make(chan int) // unbuffered

	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	// If receiver is slow, sender blocks
	time.Sleep(100 * time.Millisecond) // simulate slow receiver
	for val := range ch {
		fmt.Printf("Received: %d\n", val)
	}
}

// GOOD: Appropriately buffered channel
func goodChannelUsage() {
	ch := make(chan int, 5) // buffered for expected capacity

	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	for val := range ch {
		fmt.Printf("Received: %d\n", val)
	}
}

// Test functions
func TestGoroutineComparison(t *testing.T) {
	t.Run("Bad Goroutine Example", func(t *testing.T) {
		start := time.Now()
		badGoroutineExample()
		duration := time.Since(start)
		t.Logf("Bad example took: %v", duration)
	})

	t.Run("Good Goroutine Example", func(t *testing.T) {
		start := time.Now()
		goodGoroutineExample()
		duration := time.Since(start)
		t.Logf("Good example took: %v", duration)
	})
}

func TestSearchComparison(t *testing.T) {
	query := "golang concurrency"

	t.Run("Sequential Search", func(t *testing.T) {
		start := time.Now()
		results := badSequentialSearch(query)
		duration := time.Since(start)
		t.Logf("Sequential search took: %v, got %d results", duration, len(results))
	})

	t.Run("Parallel Search", func(t *testing.T) {
		start := time.Now()
		results := goodParallelSearch(query)
		duration := time.Since(start)
		t.Logf("Parallel search took: %v, got %d results", duration, len(results))
	})

	t.Run("Parallel Search with Timeout", func(t *testing.T) {
		start := time.Now()
		results := betterParallelSearchWithTimeout(query)
		duration := time.Since(start)
		t.Logf("Parallel search with timeout took: %v, got %d results", duration, len(results))
	})
}

func TestWorkerPoolComparison(t *testing.T) {
	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{ID: i, Data: fmt.Sprintf("job-%d", i)}
	}

	t.Run("Bad Job Processing", func(t *testing.T) {
		start := time.Now()
		results := badJobProcessing(jobs)
		duration := time.Since(start)
		t.Logf("Bad job processing took: %v, processed %d jobs", duration, len(results))
	})

	t.Run("Good Worker Pool", func(t *testing.T) {
		start := time.Now()
		results := goodWorkerPoolProcessing(jobs)
		duration := time.Since(start)
		t.Logf("Worker pool took: %v, processed %d jobs", duration, len(results))
	})
}

func TestContextComparison(t *testing.T) {
	data := []string{"item1", "item2", "item3", "item4", "item5"}

	t.Run("Bad Long Running Operation", func(t *testing.T) {
		start := time.Now()
		results := badLongRunningOperation(data)
		duration := time.Since(start)
		t.Logf("Bad operation took: %v, processed %d items", duration, len(results))
	})

	t.Run("Good Context Aware Operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		start := time.Now()
		results, err := goodContextAwareOperation(ctx, data)
		duration := time.Since(start)

		if err != nil {
			t.Logf("Context aware operation cancelled after %v, processed %d items: %v", duration, len(results), err)
		} else {
			t.Logf("Context aware operation completed in %v, processed %d items", duration, len(results))
		}
	})
}

func TestChannelComparison(t *testing.T) {
	t.Run("Bad Channel Usage", func(t *testing.T) {
		start := time.Now()
		badChannelUsage()
		duration := time.Since(start)
		t.Logf("Bad channel usage took: %v", duration)
	})

	t.Run("Good Channel Usage", func(t *testing.T) {
		start := time.Now()
		goodChannelUsage()
		duration := time.Since(start)
		t.Logf("Good channel usage took: %v", duration)
	})
}

// Benchmark tests
func BenchmarkSequentialSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		badSequentialSearch("test")
	}
}

func BenchmarkParallelSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		goodParallelSearch("test")
	}
}

func BenchmarkBadJobProcessing(b *testing.B) {
	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{ID: i, Data: fmt.Sprintf("job-%d", i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		badJobProcessing(jobs)
	}
}

func BenchmarkGoodWorkerPool(b *testing.B) {
	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{ID: i, Data: fmt.Sprintf("job-%d", i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goodWorkerPoolProcessing(jobs)
	}
}
