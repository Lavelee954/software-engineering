# Go Concurrency Patterns: Comprehensive Programming Guide

This guide demonstrates 18 progressive Go concurrency patterns, from basic goroutine concepts to production-ready systems. Each pattern builds upon previous concepts while introducing essential techniques for mastering Go concurrency.

Go follows the philosophy of **"Don't communicate by sharing memory; share memory by communicating"** - using message passing through channels instead of traditional shared memory synchronization.

## Pattern Learning Progression

### Foundation Patterns (Examples 1-3)
**Building blocks: Goroutines, Channels, and Generators**

### Coordination Patterns (Examples 4-8) 
**Advanced channel techniques: Fan-in, Sequencing, Timeouts, Shutdown**

### Real-World Application (Examples 9-12)
**Concurrent search evolution: Sequential → Parallel → Timeout → Replicated**

### Production Patterns (Examples 13-18)
**Advanced systems: Pub/Sub, Worker Pools, Context, Ring Buffers**

---

## Core Programming Guidelines

### 1. Foundation Patterns (1-3): Basic Goroutines and Channels

**Example 1: Basic Goroutines (1-boring/)**
```go
// Launch goroutines for concurrent execution
go boring("boring!")

// Key lesson: main goroutine doesn't wait for spawned goroutines
// Solution: Use channels for synchronization
```

**Example 2: Channel Communication (2-chan/)**
```go
// Create channel for goroutine communication
c := make(chan string)
go boring("boring!", c)

// Synchronous receive blocks until data is available
msg := <-c
```

**Example 3: Generator Pattern (3-generator/)**
```go
// Function returns receive-only channel
func boring(msg string) <-chan string {
    c := make(chan string)
    go func() {
        for i := 0; ; i++ {
            c <- fmt.Sprintf("%s %d", msg, i)
            time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
        }
    }()
    return c
}

// Use generator to create data streams
joe := boring("Joe")
ann := boring("Ann")
```

### 2. Coordination Patterns (4-8): Advanced Channel Techniques

**Example 4: Fan-In Pattern (4-fanin/)**
```go
// Multiplex multiple channels into one
func fanIn(input1, input2 <-chan string) <-chan string {
    c := make(chan string)
    go func() { for { c <- <-input1 } }()
    go func() { for { c <- <-input2 } }()
    return c
}

// Variadic version for multiple inputs
func fanInVariadic(inputs ...<-chan string) <-chan string {
    c := make(chan string)
    for _, input := range inputs {
        input := input
        go func() { for { c <- <-input } }()
    }
    return c
}
```

**Example 5: Sequencing with Wait Channels (5-restore-sequence/)**
```go
// Message with embedded wait channel for ordering
type Message struct {
    str  string
    wait chan bool
}

// Maintain order while preserving concurrency
func fanInWithSequence(input1, input2 <-chan Message) <-chan Message {
    c := make(chan Message)
    go func() { for { c <- <-input1 } }()
    go func() { for { c <- <-input2 } }()
    return c
}
```

**Example 6: Timeout Pattern (6-select-timeout/)**
```go
// Use select with timeout to prevent blocking
select {
case s := <-c:
    fmt.Println(s)
case <-time.After(1 * time.Second):
    fmt.Println("You're too slow.")
    return
}
```

**Example 7: Quit Signals (7-quit-signal/)**
```go
// Graceful shutdown with bidirectional signaling
quit := make(chan bool)
c := boring("Joe", quit)

for i := rand.Intn(10); i >= 0; i-- {
    fmt.Println(<-c)
}
quit <- true
```

**Example 8: Daisy Chain (8-daisy-chan/)**
```go
// Create chain of goroutines for sequential processing
const n = 1000
leftmost := make(chan int)
right := leftmost
left := leftmost

for i := 0; i < n; i++ {
    right = make(chan int)
    go f(left, right)
    left = right
}
```

### 3. Real-World Application (9-12): Google Search Evolution

**Example 9: Sequential Search (9-google1.0/)**
```go
// Baseline: Sequential execution
func Google(query string) (results []Result) {
    results = append(results, Web(query))
    results = append(results, Image(query))
    results = append(results, Video(query))
    return
}
```

**Example 10: Parallel Search (10-google2.0/)**
```go
// Improved: Parallel execution with goroutines
func Google(query string) (results []Result) {
    c := make(chan Result)
    go func() { c <- Web(query) }()
    go func() { c <- Image(query) }()
    go func() { c <- Video(query) }()
    
    for i := 0; i < 3; i++ {
        result := <-c
        results = append(results, result)
    }
    return
}
```

**Example 11: Search with Timeout (11-google2.1/)**
```go
// Enhanced: Add timeout for responsiveness
func Google(query string) (results []Result) {
    c := make(chan Result)
    go func() { c <- Web(query) }()
    go func() { c <- Image(query) }()
    go func() { c <- Video(query) }()
    
    timeout := time.After(50 * time.Millisecond)
    for i := 0; i < 3; i++ {
        select {
        case result := <-c:
            results = append(results, result)
        case <-timeout:
            fmt.Println("timed out")
            return
        }
    }
    return
}
```

**Example 12: Replicated Search (12-google3.0/)**
```go
// Advanced: Multiple replicas with first-response wins
func First(query string, replicas ...Search) Result {
    c := make(chan Result)
    searchReplica := func(i int) { c <- replicas[i](query) }
    for i := range replicas {
        go searchReplica(i)
    }
    return <-c
}

func Google(query string) (results []Result) {
    c := make(chan Result)
    go func() { c <- First(query, Web1, Web2) }()
    go func() { c <- First(query, Image1, Image2) }()
    go func() { c <- First(query, Video1, Video2) }()
    // ... with timeout logic
}
```

### 4. Production Patterns (13-18): Advanced Systems

**Example 13: Ping-Pong (13-adv-pingpong/)**
```go
// Alternating communication pattern
type Ball struct{ hits int }

func player(name string, table chan *Ball) {
    for {
        ball := <-table
        ball.hits++
        fmt.Println(name, ball.hits)
        time.Sleep(100 * time.Millisecond)
        table <- ball
    }
}
```

**Example 14: Subscription System (14-adv-subscription/)**
```go
// Complex pub/sub with fetching, deduplication, merging
type Subscription interface {
    Updates() <-chan Item
    Close() error
}

func Merge(subs ...Subscription) <-chan Item {
    out := make(chan Item)
    go func() {
        defer close(out)
        // Sophisticated merging logic with deduplication
    }()
    return out
}
```

**Example 15: Bounded Parallelism (15-bounded-parallelism/)**
```go
// Control concurrency with worker pools
func MD5All(root string) (map[string][md5.Size]byte, error) {
    done := make(chan struct{})
    defer close(done)
    
    paths, errc := walkFiles(done, root)
    
    // Start fixed number of goroutines
    c := make(chan result)
    var wg sync.WaitGroup
    const numDigesters = 20
    wg.Add(numDigesters)
    
    for i := 0; i < numDigesters; i++ {
        go func() {
            digester(done, paths, c)
            wg.Done()
        }()
    }
    
    go func() {
        wg.Wait()
        close(c)
    }()
    
    // Collect results
    m := make(map[string][md5.Size]byte)
    for r := range c {
        if r.err != nil {
            return nil, r.err
        }
        m[r.path] = r.sum
    }
    return m, nil
}
```

**Example 16: Context Usage (16-context/)**
```go
// Request scoping and cancellation
func handleSearch(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()
    ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
    defer cancel()
    
    query := req.FormValue("q")
    results, err := google.Search(ctx, query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    render(w, results)
}
```

**Example 17: Ring Buffer (17-ring-buffer-channel/)**
```go
// Bounded buffer with overflow protection
type RingBuffer struct {
    buffer chan interface{}
}

func (rb *RingBuffer) Send(item interface{}) {
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
```

**Example 18: Worker Pool (18-worker-pool/)**
```go
// Efficient worker pool implementation
func worker(id int, jobs <-chan Job, results chan<- Result) {
    for j := range jobs {
        fmt.Printf("worker %d started job %d\n", id, j.ID)
        time.Sleep(time.Second)
        fmt.Printf("worker %d finished job %d\n", id, j.ID)
        results <- Result{j.ID, j.ID * 2}
    }
}

// Nested goroutines for efficiency
func efficientWorkerPool(jobs []Job) []Result {
    jobsChan := make(chan Job, len(jobs))
    resultsChan := make(chan Result, len(jobs))
    
    const numWorkers = 3
    var wg sync.WaitGroup
    
    // Start workers
    for w := 1; w <= numWorkers; w++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := range jobsChan {
                // Process job
                resultsChan <- Result{j.ID, j.ID * 2}
            }
        }(w)
    }
    
    // Send jobs
    go func() {
        for _, job := range jobs {
            jobsChan <- job
        }
        close(jobsChan)
    }()
    
    // Close results channel when all workers done
    go func() {
        wg.Wait()
        close(resultsChan)
    }()
    
    // Collect results
    var results []Result
    for r := range resultsChan {
        results = append(results, r)
    }
    
    return results
}
```

## Best Practices Summary

### Channel Design Principles
1. **Unbuffered channels**: For synchronous communication and signaling
2. **Buffered channels**: For throughput optimization and backpressure control
3. **Receive-only channels** (`<-chan T`): For generator patterns and API safety
4. **Send-only channels** (`chan<- T`): For producer interfaces

### Goroutine Management
1. **Always plan goroutine lifecycle**: Have clear start and termination conditions
2. **Use context for cancellation**: Propagate cancellation through call stacks
3. **Implement graceful shutdown**: Use quit channels and sync.WaitGroup
4. **Avoid goroutine leaks**: Ensure all goroutines can terminate

### Error Handling Patterns
1. **Error channels**: Send errors through dedicated channels
2. **Context with timeout**: Prevent indefinite blocking
3. **First error wins**: Use errgroup for coordinated error handling
4. **Partial results**: Design for graceful degradation

### Performance Considerations
1. **Bounded parallelism**: Control resource usage with worker pools
2. **Channel buffering**: Size buffers based on producer/consumer rates
3. **Ring buffers**: For high-throughput scenarios with overflow protection
4. **Profiling**: Use go tool pprof and trace for optimization

### Testing Concurrent Code
1. **Race detector**: Always run `go test -race`
2. **Deterministic tests**: Use channels for synchronization in tests
3. **Timeout tests**: Prevent tests from hanging indefinitely
4. **Stress testing**: Test with high concurrency loads

## Pattern Selection Guide

- **Simple parallel execution**: Use basic goroutines (Examples 1-2)
- **Data generation**: Generator pattern (Example 3)
- **Merging data streams**: Fan-in pattern (Example 4)
- **Ordered processing**: Sequencing with wait channels (Example 5)
- **Responsive systems**: Timeout patterns (Examples 6, 11)
- **Graceful shutdown**: Quit signals (Example 7)
- **Sequential processing**: Daisy chain (Example 8)
- **Real-world services**: Google search evolution (Examples 9-12)
- **Turn-based systems**: Ping-pong (Example 13)
- **Event streaming**: Subscription systems (Example 14)
- **Resource control**: Bounded parallelism (Example 15)
- **Request lifecycle**: Context patterns (Example 16)
- **Overflow handling**: Ring buffers (Example 17)
- **Load balancing**: Worker pools (Example 18)

Each pattern solves specific concurrent programming challenges while demonstrating Go's channel-based approach to building reliable, efficient concurrent systems.