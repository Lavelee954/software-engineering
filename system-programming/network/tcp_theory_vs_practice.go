package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// TheoreticalTCPConnection represents the ideal TCP connection behavior
type TheoreticalTCPConnection struct {
	reliable     bool
	ordered      bool
	flowControl  bool
	errorChecked bool
}

// PracticalTCPConnection represents real-world TCP connection with all the complexities
type PracticalTCPConnection struct {
	conn              net.Conn
	reader            *bufio.Reader
	writer            *bufio.Writer
	connected         bool
	lastError         error
	retryCount        int
	maxRetries        int
	timeout           time.Duration
	mu                sync.RWMutex
	reconnectAttempts int
}

// TheoryReflection shows what we expect vs what actually happens
type TheoryReflection struct {
	ExpectedBehavior string
	ActualBehavior   string
	Lesson           string
}

func NewPracticalTCPConnection(address string, timeout time.Duration) *PracticalTCPConnection {
	return &PracticalTCPConnection{
		maxRetries: 3,
		timeout:    timeout,
	}
}

// Connect demonstrates the difference between theoretical "just connect" and practical reality
func (p *PracticalTCPConnection) Connect(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Theory: TCP connection is established immediately
	// Practice: Network delays, timeouts, retries needed
	
	var conn net.Conn
	var err error
	
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		// Practical consideration: Add exponential backoff
		if attempt > 0 {
			backoff := time.Duration(attempt) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
		
		// Practical consideration: Use context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		dialer := &net.Dialer{}
		
		conn, err = dialer.DialContext(ctx, "tcp", address)
		cancel()
		
		if err == nil {
			break
		}
		
		p.reconnectAttempts++
		log.Printf("Connection attempt %d failed: %v", attempt+1, err)
	}
	
	if err != nil {
		p.lastError = fmt.Errorf("failed to connect after %d attempts: %w", p.maxRetries+1, err)
		return p.lastError
	}
	
	p.conn = conn
	p.reader = bufio.NewReader(conn)
	p.writer = bufio.NewWriter(conn)
	p.connected = true
	
	return nil
}

// Send demonstrates reliable delivery complexities
func (p *PracticalTCPConnection) Send(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.connected {
		return fmt.Errorf("not connected")
	}
	
	// Theory: TCP guarantees delivery, so just send
	// Practice: Need to handle partial writes, timeouts, connection drops
	
	// Set write deadline for timeout handling
	p.conn.SetWriteDeadline(time.Now().Add(p.timeout))
	
	totalWritten := 0
	for totalWritten < len(data) {
		written, err := p.writer.Write(data[totalWritten:])
		if err != nil {
			// Practical consideration: Connection might be broken
			if isConnectionError(err) {
				p.connected = false
				return fmt.Errorf("connection lost during send: %w", err)
			}
			return fmt.Errorf("write error: %w", err)
		}
		
		totalWritten += written
		
		// Practical consideration: Flush buffer periodically
		if totalWritten%1024 == 0 {
			if err := p.writer.Flush(); err != nil {
				return fmt.Errorf("flush error: %w", err)
			}
		}
	}
	
	// Ensure all data is sent
	if err := p.writer.Flush(); err != nil {
		return fmt.Errorf("final flush error: %w", err)
	}
	
	return nil
}

// Receive demonstrates ordered delivery and buffering realities
func (p *PracticalTCPConnection) Receive(buffer []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.connected {
		return 0, fmt.Errorf("not connected")
	}
	
	// Theory: TCP delivers data in order, so just read
	// Practice: Handle partial reads, timeouts, connection drops, buffering
	
	// Set read deadline for timeout handling
	p.conn.SetReadDeadline(time.Now().Add(p.timeout))
	
	totalRead := 0
	for totalRead < len(buffer) {
		n, err := p.reader.Read(buffer[totalRead:])
		if err != nil {
			if err == io.EOF {
				// Practical consideration: Connection closed by peer
				p.connected = false
				if totalRead > 0 {
					return totalRead, nil // Return partial data
				}
				return 0, err
			}
			
			if isConnectionError(err) {
				p.connected = false
				return totalRead, fmt.Errorf("connection lost during receive: %w", err)
			}
			
			// Practical consideration: Timeout might be acceptable
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if totalRead > 0 {
					return totalRead, nil // Return partial data on timeout
				}
			}
			
			return totalRead, fmt.Errorf("read error: %w", err)
		}
		
		totalRead += n
		
		// Practical consideration: Check if we have a complete message
		// In real applications, you'd implement message framing
		if totalRead > 0 && buffer[totalRead-1] == '\n' {
			break // Complete line received
		}
	}
	
	return totalRead, nil
}

// Close demonstrates proper connection teardown
func (p *PracticalTCPConnection) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.connected || p.conn == nil {
		return nil
	}
	
	// Theory: Just close the connection
	// Practice: Graceful shutdown, handle errors, cleanup resources
	
	// Practical consideration: Flush any pending writes
	if p.writer != nil {
		p.writer.Flush()
	}
	
	// Practical consideration: Set deadline for close operation
	p.conn.SetDeadline(time.Now().Add(5 * time.Second))
	
	err := p.conn.Close()
	p.connected = false
	p.conn = nil
	p.reader = nil
	p.writer = nil
	
	return err
}

// GetStats returns connection statistics for analysis
func (p *PracticalTCPConnection) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return map[string]interface{}{
		"connected":           p.connected,
		"retry_count":         p.retryCount,
		"reconnect_attempts":  p.reconnectAttempts,
		"last_error":          p.lastError,
		"timeout":             p.timeout,
	}
}

// isConnectionError checks if error indicates connection problem
func isConnectionError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return !netErr.Timeout()
	}
	return false
}

// GenerateTheoryReflections provides insights into theory vs practice
func GenerateTheoryReflections() []TheoryReflection {
	return []TheoryReflection{
		{
			ExpectedBehavior: "TCP connection establishes immediately when calling connect()",
			ActualBehavior:   "Connection attempts may fail due to network issues, timeouts, or server unavailability",
			Lesson:           "Always implement retry logic with exponential backoff and proper timeout handling",
		},
		{
			ExpectedBehavior: "TCP guarantees reliable delivery, so data will always arrive",
			ActualBehavior:   "Connection can drop mid-transmission, requiring detection and handling",
			Lesson:           "Monitor connection state and implement reconnection strategies",
		},
		{
			ExpectedBehavior: "TCP preserves message boundaries automatically",
			ActualBehavior:   "TCP is a stream protocol - it doesn't preserve message boundaries",
			Lesson:           "Implement application-level message framing and buffering",
		},
		{
			ExpectedBehavior: "Writing data means it's immediately sent to the receiver",
			ActualBehavior:   "Data may be buffered, partially sent, or delayed due to flow control",
			Lesson:           "Handle partial writes and implement proper flushing mechanisms",
		},
		{
			ExpectedBehavior: "Reading data returns complete messages",
			ActualBehavior:   "Reads may return partial data requiring multiple read operations",
			Lesson:           "Implement read loops and message assembly logic",
		},
		{
			ExpectedBehavior: "TCP handles all error recovery automatically",
			ActualBehavior:   "Application must handle connection errors, timeouts, and recovery",
			Lesson:           "Implement comprehensive error handling and recovery strategies",
		},
	}
}

// DemonstrateTCPRealities shows practical TCP programming challenges
func DemonstrateTCPRealities() {
	fmt.Println("=== TCP Theory vs Practice Demonstration ===")
	fmt.Println()
	
	// Show theoretical expectations vs practical realities
	reflections := GenerateTheoryReflections()
	for i, reflection := range reflections {
		fmt.Printf("Lesson %d:\n", i+1)
		fmt.Printf("Theory: %s\n", reflection.ExpectedBehavior)
		fmt.Printf("Practice: %s\n", reflection.ActualBehavior)
		fmt.Printf("Lesson: %s\n\n", reflection.Lesson)
	}
}

func DemonstrateTCPRealitiesMain() {
	DemonstrateTCPRealities()
}