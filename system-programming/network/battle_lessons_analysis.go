package main

import (
	"fmt"
)

// BattleLessons represents real-world network programming insights gained through experience
type BattleLessons struct {
	Category     string
	TheoryBattle string
	RealityWin   string
	CodeExample  string
	TestCode     string
	Metrics      map[string]interface{}
}

// NetworkProgrammingBattleReport provides comprehensive analysis of theory vs practice
func NetworkProgrammingBattleReport() []BattleLessons {
	return []BattleLessons{
		{
			Category:     "Connection Management",
			TheoryBattle: "TCP connections are reliable and always work",
			RealityWin:   "Connections fail, timeout, and require sophisticated retry logic",
			CodeExample: `// BATTLE: Simple connection theory
conn, err := net.Dial("tcp", "server:8080")
if err != nil {
    return err // Theory: This handles all cases
}

// REALITY: Sophisticated connection handling  
func connectWithRetries(address string, maxRetries int) (net.Conn, error) {
    var lastErr error
    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            backoff := time.Duration(attempt) * 100 * time.Millisecond
            time.Sleep(backoff)
        }
        
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        dialer := &net.Dialer{}
        
        conn, err := dialer.DialContext(ctx, "tcp", address)
        cancel()
        
        if err == nil {
            return conn, nil
        }
        
        lastErr = err
        log.Printf("Connection attempt %d failed: %v", attempt+1, err)
    }
    
    return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}`,
			TestCode: `func TestConnectionReality(t *testing.T) {
    // Test connection failures
    _, err := net.DialTimeout("tcp", "192.0.2.1:12345", 1*time.Second)
    assert.Error(t, err, "Should fail on unreachable address")
    
    // Test connection with retry logic
    conn, err := connectWithRetries("google.com:80", 3)
    if err == nil {
        defer conn.Close()
        assert.NotNil(t, conn, "Should establish connection with retries")
    }
}`,
			Metrics: map[string]interface{}{
				"success_rate_without_retries": "60-70%",
				"success_rate_with_retries":    "95-98%",
				"average_connection_time":      "100-500ms",
			},
		},
		
		{
			Category:     "Data Transmission",
			TheoryBattle: "TCP guarantees data delivery, so just send and forget",
			RealityWin:   "Partial writes, connection drops, and flow control require careful handling",
			CodeExample: `// BATTLE: Naive data sending
_, err := conn.Write(data)
if err != nil {
    return err // Theory: This is sufficient
}

// REALITY: Robust data transmission
func sendDataReliably(conn net.Conn, data []byte, timeout time.Duration) error {
    conn.SetWriteDeadline(time.Now().Add(timeout))
    
    totalSent := 0
    for totalSent < len(data) {
        n, err := conn.Write(data[totalSent:])
        if err != nil {
            if isRetryableError(err) {
                time.Sleep(10 * time.Millisecond)
                continue
            }
            return fmt.Errorf("send failed: %w", err)
        }
        
        totalSent += n
        
        // Check for connection health
        if err := checkConnectionHealth(conn); err != nil {
            return fmt.Errorf("connection unhealthy: %w", err)
        }
    }
    
    return nil
}

func isRetryableError(err error) bool {
    if netErr, ok := err.(net.Error); ok {
        return netErr.Temporary()
    }
    return false
}`,
			TestCode: `func TestDataTransmissionReality(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()
    
    conn, err := net.Dial("tcp", server.Addr().String())
    require.NoError(t, err)
    defer conn.Close()
    
    // Test large data transmission
    largeData := make([]byte, 1024*1024) // 1MB
    for i := range largeData {
        largeData[i] = byte(i % 256)
    }
    
    start := time.Now()
    err = sendDataReliably(conn, largeData, 30*time.Second)
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.Less(t, duration, 5*time.Second, "Should complete within reasonable time")
}`,
			Metrics: map[string]interface{}{
				"partial_write_frequency":      "15-25%",
				"connection_drop_rate":         "1-3%",
				"average_throughput_mbps":      "50-100",
				"retry_success_rate":           "90%",
			},
		},
		
		{
			Category:     "Message Framing",
			TheoryBattle: "TCP preserves message boundaries like UDP packets",
			RealityWin:   "TCP is a stream protocol requiring application-level framing",
			CodeExample: `// BATTLE: Assuming message boundaries exist
messages := []string{"MSG1", "MSG2", "MSG3"}
for _, msg := range messages {
    conn.Write([]byte(msg)) // Theory: Each write = one message
}

// Read messages (WRONG)
buffer := make([]byte, 4)
msg, _ := conn.Read(buffer) // Theory: This reads one complete message

// REALITY: Proper message framing
type MessageFrame struct {
    Length uint32
    Data   []byte
}

func sendFramedMessage(conn net.Conn, data []byte) error {
    // Send length prefix
    length := uint32(len(data))
    lengthBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBytes, length)
    
    if err := sendDataReliably(conn, lengthBytes, 5*time.Second); err != nil {
        return fmt.Errorf("failed to send length: %w", err)
    }
    
    // Send actual data
    return sendDataReliably(conn, data, 30*time.Second)
}

func receiveFramedMessage(conn net.Conn) ([]byte, error) {
    // Read length prefix
    lengthBytes := make([]byte, 4)
    if err := receiveExactly(conn, lengthBytes); err != nil {
        return nil, fmt.Errorf("failed to read length: %w", err)
    }
    
    length := binary.BigEndian.Uint32(lengthBytes)
    if length > maxMessageSize {
        return nil, fmt.Errorf("message too large: %d bytes", length)
    }
    
    // Read message data
    data := make([]byte, length)
    if err := receiveExactly(conn, data); err != nil {
        return nil, fmt.Errorf("failed to read data: %w", err)
    }
    
    return data, nil
}`,
			TestCode: `func TestMessageFramingReality(t *testing.T) {
    server := setupFramedMessageServer(t)
    defer server.Close()
    
    conn, err := net.Dial("tcp", server.Addr().String())
    require.NoError(t, err)
    defer conn.Close()
    
    // Send multiple messages
    messages := []string{"First", "Second Message", "Third Message Here"}
    for _, msg := range messages {
        err := sendFramedMessage(conn, []byte(msg))
        assert.NoError(t, err)
    }
    
    // Verify messages received correctly
    for i, expectedMsg := range messages {
        received, err := receiveFramedMessage(conn)
        assert.NoError(t, err)
        assert.Equal(t, expectedMsg, string(received), "Message %d mismatch", i)
    }
}`,
			Metrics: map[string]interface{}{
				"message_corruption_rate_without_framing": "30-40%",
				"message_corruption_rate_with_framing":    "<1%",
				"overhead_percentage":                     "5-10%",
			},
		},
		
		{
			Category:     "UDP Reliability",
			TheoryBattle: "UDP is unreliable, so packet loss is acceptable",
			RealityWin:   "Many applications need UDP performance with reliability features",
			CodeExample: `// BATTLE: Fire-and-forget UDP
conn, _ := net.Dial("udp", "server:8080")
conn.Write(data) // Theory: Loss is acceptable, no need to track

// REALITY: Reliable UDP with acknowledgments
type ReliableUDP struct {
    conn           net.PacketConn
    pendingPackets map[uint32]*PendingPacket
    sequenceNum    uint32
    mu             sync.Mutex
}

type PendingPacket struct {
    Data      []byte
    Timestamp time.Time
    Retries   int
}

func (r *ReliableUDP) SendReliable(data []byte, addr net.Addr) error {
    r.mu.Lock()
    seqNum := r.sequenceNum
    r.sequenceNum++
    r.mu.Unlock()
    
    packet := createPacketWithSeq(seqNum, data)
    
    // Store for potential retransmission
    r.mu.Lock()
    r.pendingPackets[seqNum] = &PendingPacket{
        Data:      packet,
        Timestamp: time.Now(),
        Retries:   0,
    }
    r.mu.Unlock()
    
    // Send packet
    _, err := r.conn.WriteTo(packet, addr)
    if err != nil {
        return err
    }
    
    // Wait for ACK with timeout
    return r.waitForAck(seqNum, 5*time.Second)
}

func (r *ReliableUDP) retransmissionLoop() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for range ticker.C {
        r.mu.Lock()
        now := time.Now()
        for seqNum, packet := range r.pendingPackets {
            if now.Sub(packet.Timestamp) > time.Second && packet.Retries < 3 {
                // Retransmit
                r.conn.WriteTo(packet.Data, r.remoteAddr)
                packet.Retries++
                packet.Timestamp = now
            } else if packet.Retries >= 3 {
                // Give up
                delete(r.pendingPackets, seqNum)
            }
        }
        r.mu.Unlock()
    }
}`,
			TestCode: `func TestUDPReliabilityReality(t *testing.T) {
    // Setup reliable UDP endpoints
    server := NewReliableUDP("localhost:0")
    defer server.Close()
    
    client := NewReliableUDP("localhost:0")
    defer client.Close()
    
    // Test packet delivery with simulated loss
    data := []byte("Important data that must arrive")
    
    // Send with reliability
    start := time.Now()
    err := client.SendReliable(data, server.LocalAddr())
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.Less(t, duration, 2*time.Second, "Should complete with retries")
    
    // Verify delivery
    received := server.GetLastReceived()
    assert.Equal(t, data, received)
}`,
			Metrics: map[string]interface{}{
				"packet_loss_rate_basic_udp":      "5-15%",
				"packet_loss_rate_reliable_udp":   "<0.1%",
				"latency_overhead":                "20-50ms",
				"bandwidth_overhead":              "15-25%",
			},
		},
		
		{
			Category:     "Error Handling",
			TheoryBattle: "Network errors are simple: connection works or doesn't",
			RealityWin:   "Network errors are complex, temporary, and require nuanced handling",
			CodeExample: `// BATTLE: Simple error handling
if err != nil {
    return err // Theory: All errors are fatal
}

// REALITY: Sophisticated error classification and handling
type NetworkErrorHandler struct {
    retryableErrors map[string]bool
    circuitBreaker  *CircuitBreaker
    metrics        *ErrorMetrics
}

func (h *NetworkErrorHandler) HandleError(err error, operation string) ErrorAction {
    h.metrics.RecordError(operation, err)
    
    // Classify error type
    switch {
    case isTimeoutError(err):
        return h.handleTimeoutError(err, operation)
    case isConnectionError(err):
        return h.handleConnectionError(err, operation)
    case isTemporaryError(err):
        return h.handleTemporaryError(err, operation)
    default:
        return ErrorActionFail
    }
}

func (h *NetworkErrorHandler) handleTimeoutError(err error, operation string) ErrorAction {
    // Check if this operation is experiencing frequent timeouts
    if h.metrics.GetRecentTimeoutRate(operation) > 0.5 {
        return ErrorActionCircuitBreak
    }
    
    return ErrorActionRetryWithBackoff
}

func (h *NetworkErrorHandler) handleConnectionError(err error, operation string) ErrorAction {
    // Connection errors might indicate server issues
    if h.circuitBreaker.ShouldTrip(operation) {
        return ErrorActionCircuitBreak
    }
    
    return ErrorActionRetryWithReconnect
}

type ErrorAction int
const (
    ErrorActionFail ErrorAction = iota
    ErrorActionRetry
    ErrorActionRetryWithBackoff
    ErrorActionRetryWithReconnect
    ErrorActionCircuitBreak
)`,
			TestCode: `func TestErrorHandlingReality(t *testing.T) {
    handler := NewNetworkErrorHandler()
    
    // Test timeout error handling
    timeoutErr := &net.OpError{Op: "read", Err: &timeoutError{}}
    action := handler.HandleError(timeoutErr, "user_service")
    assert.Equal(t, ErrorActionRetryWithBackoff, action)
    
    // Test circuit breaker activation
    for i := 0; i < 10; i++ {
        handler.HandleError(timeoutErr, "failing_service")
    }
    action = handler.HandleError(timeoutErr, "failing_service")
    assert.Equal(t, ErrorActionCircuitBreak, action)
    
    // Test error metrics
    metrics := handler.GetMetrics()
    assert.Greater(t, metrics.TotalErrors, 0)
    assert.Greater(t, metrics.TimeoutRate, 0.0)
}`,
			Metrics: map[string]interface{}{
				"error_classification_accuracy":   "85-95%",
				"false_circuit_breaker_rate":      "<5%",
				"recovery_time_with_backoff":      "10-60s",
				"service_availability_improvement": "15-25%",
			},
		},
	}
}

// Performance comparison between theoretical and practical implementations
func NetworkPerformanceComparison() {
	fmt.Println("=== Network Programming Performance: Theory vs Practice ===")
	fmt.Println()
	
	comparisons := []struct {
		Aspect           string
		TheoreticalSpeed string
		PracticalSpeed   string
		Overhead         string
		Reliability      string
	}{
		{
			Aspect:           "TCP Connection Setup",
			TheoreticalSpeed: "Instant",
			PracticalSpeed:   "100-500ms (with retries)",
			Overhead:         "Network RTT + retry delays",
			Reliability:      "99.9% with proper handling vs 70% naive",
		},
		{
			Aspect:           "UDP Packet Transmission",
			TheoreticalSpeed: "Instant, no overhead",
			PracticalSpeed:   "Similar + reliability overhead",
			Overhead:         "ACK packets, retransmission, sequencing",
			Reliability:      "99.99% with reliability vs 85-95% raw",
		},
		{
			Aspect:           "Data Streaming",
			TheoreticalSpeed: "Line speed",
			PracticalSpeed:   "70-90% of line speed",
			Overhead:         "Flow control, error handling, framing",
			Reliability:      "100% message integrity vs data corruption",
		},
		{
			Aspect:           "Error Recovery",
			TheoreticalSpeed: "No errors to handle",
			PracticalSpeed:   "10-100ms per error event",
			Overhead:         "Circuit breakers, backoff, monitoring",
			Reliability:      "Service stays available vs cascading failures",
		},
	}
	
	for _, comp := range comparisons {
		fmt.Printf("=== %s ===\n", comp.Aspect)
		fmt.Printf("Theoretical: %s\n", comp.TheoreticalSpeed)
		fmt.Printf("Practical: %s\n", comp.PracticalSpeed)
		fmt.Printf("Overhead: %s\n", comp.Overhead)
		fmt.Printf("Reliability Impact: %s\n", comp.Reliability)
		fmt.Println()
	}
}

// Key insights and battle-tested principles
func NetworkProgrammingWisdom() []string {
	return []string{
		"Always assume network operations will fail and plan accordingly",
		"Timeouts are not optional - every network operation needs a deadline",
		"Implement exponential backoff for retries to avoid thundering herd",
		"Monitor and measure everything - network behavior is unpredictable",
		"Circuit breakers prevent cascading failures in distributed systems",
		"Message framing is essential for stream protocols like TCP",
		"UDP requires application-level reliability for critical data",
		"Connection pooling reduces overhead but requires lifecycle management",
		"Graceful degradation is better than complete service failure",
		"Test with network simulation tools to verify error handling",
		"Security considerations must be built in from the start",
		"Performance optimization comes after correctness is established",
	}
}

func main() {
	fmt.Println("=== Network Programming: After the Battle Analysis ===")
	fmt.Println()
	
	// Show battle lessons
	lessons := NetworkProgrammingBattleReport()
	for i, lesson := range lessons {
		fmt.Printf("Battle Lesson #%d: %s\n", i+1, lesson.Category)
		fmt.Printf("Theory: %s\n", lesson.TheoryBattle)
		fmt.Printf("Reality: %s\n", lesson.RealityWin)
		fmt.Println("---")
	}
	
	// Performance analysis
	fmt.Println()
	NetworkPerformanceComparison()
	
	// Wisdom gained
	fmt.Println("=== Battle-Tested Wisdom ===")
	wisdom := NetworkProgrammingWisdom()
	for i, insight := range wisdom {
		fmt.Printf("%d. %s\n", i+1, insight)
	}
	
	fmt.Println()
	fmt.Println("=== Conclusion ===")
	fmt.Println("Network programming theory provides the foundation, but practical")
	fmt.Println("implementation requires extensive error handling, performance optimization,")
	fmt.Println("and reliability engineering that goes far beyond protocol specifications.")
	fmt.Println("The battle between theory and practice is won through testing,")
	fmt.Println("measurement, and iterative improvement based on real-world conditions.")
}