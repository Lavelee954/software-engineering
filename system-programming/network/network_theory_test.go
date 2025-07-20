package main

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test structures for comparing theory vs practice

type NetworkTestCase struct {
	Name                string
	TheoreticalExpected string
	PracticalReality    string
	TestFunction        func(t *testing.T) TestResult
}

type TestResult struct {
	Success           bool
	ExpectedBehavior  string
	ActualBehavior    string
	PerformanceMetrics map[string]interface{}
	Lessons           []string
}

// TCP Theory vs Practice Tests

func TestTCPTheoryVsPractice(t *testing.T) {
	testCases := []NetworkTestCase{
		{
			Name:                "TCP Connection Establishment",
			TheoreticalExpected: "Connection establishes immediately",
			PracticalReality:    "Connection may fail, timeout, or require retries",
			TestFunction:        testTCPConnectionReality,
		},
		{
			Name:                "TCP Reliable Delivery",
			TheoreticalExpected: "Data is always delivered reliably",
			PracticalReality:    "Connection can drop during transmission",
			TestFunction:        testTCPReliabilityReality,
		},
		{
			Name:                "TCP Message Boundaries",
			TheoreticalExpected: "Messages are preserved as sent",
			PracticalReality:    "TCP is a stream protocol - no message boundaries",
			TestFunction:        testTCPStreamReality,
		},
		{
			Name:                "TCP Write Operations",
			TheoreticalExpected: "Write operations complete immediately",
			PracticalReality:    "May require partial writes and buffering",
			TestFunction:        testTCPWriteReality,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result := testCase.TestFunction(t)
			
			t.Logf("=== %s ===", testCase.Name)
			t.Logf("Theory: %s", testCase.TheoreticalExpected)
			t.Logf("Practice: %s", testCase.PracticalReality)
			t.Logf("Expected: %s", result.ExpectedBehavior)
			t.Logf("Actual: %s", result.ActualBehavior)
			
			if len(result.PerformanceMetrics) > 0 {
				t.Logf("Performance Metrics:")
				for key, value := range result.PerformanceMetrics {
					t.Logf("  %s: %v", key, value)
				}
			}
			
			if len(result.Lessons) > 0 {
				t.Logf("Lessons learned:")
				for _, lesson := range result.Lessons {
					t.Logf("  - %s", lesson)
				}
			}
		})
	}
}

func testTCPConnectionReality(t *testing.T) TestResult {
	// Theory: TCP connection is established with a simple Dial
	// Practice: Need to handle timeouts, retries, and failures
	
	start := time.Now()
	attempts := 0
	var lastError error
	
	// Try connecting to a non-existent server
	for attempts < 3 {
		attempts++
		conn, err := net.DialTimeout("tcp", "192.0.2.1:12345", 1*time.Second)
		if err != nil {
			lastError = err
			time.Sleep(100 * time.Millisecond) // Backoff
			continue
		}
		conn.Close()
		break
	}
	
	duration := time.Since(start)
	
	return TestResult{
		Success:          false, // Expected failure for demo
		ExpectedBehavior: "Connection should establish immediately",
		ActualBehavior:   fmt.Sprintf("Connection failed after %d attempts: %v", attempts, lastError),
		PerformanceMetrics: map[string]interface{}{
			"total_duration":     duration,
			"connection_attempts": attempts,
			"timeout_per_attempt": "1s",
		},
		Lessons: []string{
			"Always implement connection timeouts",
			"Use exponential backoff for retries",
			"Handle network unreachability gracefully",
		},
	}
}

func testTCPReliabilityReality(t *testing.T) TestResult {
	// Theory: TCP guarantees delivery
	// Practice: Connection can be dropped, requiring detection and handling
	
	// Set up a test server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	
	serverAddr := listener.Addr().String()
	
	// Server that accepts connection then immediately closes
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond) // Brief connection
		conn.Close() // Simulate connection drop
	}()
	
	// Client attempts to send data
	client, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	
	// Try to send data
	data := []byte("Hello, World!")
	_, writeErr := client.Write(data)
	
	// Try to read response
	buffer := make([]byte, 1024)
	_, readErr := client.Read(buffer)
	
	return TestResult{
		Success:          false,
		ExpectedBehavior: "Data should be sent and response received reliably",
		ActualBehavior:   fmt.Sprintf("Write error: %v, Read error: %v", writeErr, readErr),
		PerformanceMetrics: map[string]interface{}{
			"data_size": len(data),
		},
		Lessons: []string{
			"Always check for connection errors during I/O",
			"Implement heartbeat or keepalive mechanisms",
			"Handle connection drops gracefully",
		},
	}
}

func testTCPStreamReality(t *testing.T) TestResult {
	// Theory: TCP preserves message boundaries
	// Practice: TCP is a stream protocol
	
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	
	var receivedData []string
	var mu sync.Mutex
	
	// Server that reads data in small chunks
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		
		buffer := make([]byte, 5) // Small buffer to demonstrate streaming
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				break
			}
			
			mu.Lock()
			receivedData = append(receivedData, string(buffer[:n]))
			mu.Unlock()
		}
	}()
	
	// Client sends multiple "messages"
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	
	messages := []string{"MSG1", "MSG2", "MSG3"}
	for _, msg := range messages {
		client.Write([]byte(msg))
		time.Sleep(10 * time.Millisecond) // Small delay between sends
	}
	
	time.Sleep(100 * time.Millisecond) // Allow server to read
	
	mu.Lock()
	actualReceived := strings.Join(receivedData, "|")
	mu.Unlock()
	
	return TestResult{
		Success:          true,
		ExpectedBehavior: "Three separate messages: MSG1, MSG2, MSG3",
		ActualBehavior:   fmt.Sprintf("Stream chunks: %s", actualReceived),
		PerformanceMetrics: map[string]interface{}{
			"messages_sent":      len(messages),
			"chunks_received":    len(receivedData),
			"total_data_sent":    12, // MSG1+MSG2+MSG3
		},
		Lessons: []string{
			"TCP doesn't preserve message boundaries",
			"Implement application-level framing",
			"Use delimiters or length prefixes for messages",
		},
	}
}

func testTCPWriteReality(t *testing.T) TestResult {
	// Theory: Write operations complete immediately
	// Practice: May require partial writes and flow control
	
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	
	// Server with slow reading to trigger flow control
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		
		buffer := make([]byte, 1024)
		for {
			_, err := conn.Read(buffer)
			if err != nil {
				break
			}
			time.Sleep(50 * time.Millisecond) // Slow reading
		}
	}()
	
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	
	// Send large amount of data to trigger buffering
	largeData := bytes.Repeat([]byte("A"), 64*1024) // 64KB
	
	start := time.Now()
	totalWritten := 0
	writeCount := 0
	
	for totalWritten < len(largeData) {
		n, err := client.Write(largeData[totalWritten:])
		if err != nil {
			break
		}
		totalWritten += n
		writeCount++
		
		if writeCount > 1 {
			// Partial write occurred
			break
		}
	}
	
	writeDuration := time.Since(start)
	
	return TestResult{
		Success:          true,
		ExpectedBehavior: "Large data written in single operation",
		ActualBehavior:   fmt.Sprintf("Required %d write operations, %d bytes written in %v", writeCount, totalWritten, writeDuration),
		PerformanceMetrics: map[string]interface{}{
			"data_size":        len(largeData),
			"bytes_written":    totalWritten,
			"write_operations": writeCount,
			"write_duration":   writeDuration,
		},
		Lessons: []string{
			"Handle partial writes in loops",
			"Monitor network buffer capacity",
			"Implement flow control at application level",
		},
	}
}

// UDP Theory vs Practice Tests

func TestUDPTheoryVsPractice(t *testing.T) {
	testCases := []NetworkTestCase{
		{
			Name:                "UDP Packet Delivery",
			TheoreticalExpected: "Packets are delivered immediately",
			PracticalReality:    "Packets can be lost, duplicated, or reordered",
			TestFunction:        testUDPPacketDeliveryReality,
		},
		{
			Name:                "UDP Ordering",
			TheoreticalExpected: "No ordering guarantees",
			PracticalReality:    "Out-of-order delivery is common",
			TestFunction:        testUDPOrderingReality,
		},
		{
			Name:                "UDP Performance",
			TheoreticalExpected: "Always faster than TCP",
			PracticalReality:    "Performance depends on application-level features",
			TestFunction:        testUDPPerformanceReality,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result := testCase.TestFunction(t)
			
			t.Logf("=== %s ===", testCase.Name)
			t.Logf("Theory: %s", testCase.TheoreticalExpected)
			t.Logf("Practice: %s", testCase.PracticalReality)
			t.Logf("Expected: %s", result.ExpectedBehavior)
			t.Logf("Actual: %s", result.ActualBehavior)
			
			if len(result.PerformanceMetrics) > 0 {
				t.Logf("Performance Metrics:")
				for key, value := range result.PerformanceMetrics {
					t.Logf("  %s: %v", key, value)
				}
			}
		})
	}
}

func testUDPPacketDeliveryReality(t *testing.T) TestResult {
	// Theory: UDP packets are delivered independently
	// Practice: Packets can be lost or duplicated
	
	// Set up UDP server
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	
	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	
	actualServerAddr := server.LocalAddr().String()
	
	packetsReceived := 0
	go func() {
		buffer := make([]byte, 1024)
		for {
			_, _, err := server.ReadFromUDP(buffer)
			if err != nil {
				break
			}
			packetsReceived++
		}
	}()
	
	// Send packets from client
	client, err := net.Dial("udp", actualServerAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	
	packetsSent := 100
	for i := 0; i < packetsSent; i++ {
		data := fmt.Sprintf("packet-%d", i)
		client.Write([]byte(data))
		time.Sleep(1 * time.Millisecond) // Small delay
	}
	
	time.Sleep(100 * time.Millisecond) // Wait for packets to arrive
	
	deliveryRate := float64(packetsReceived) / float64(packetsSent) * 100
	
	return TestResult{
		Success:          true,
		ExpectedBehavior: "All 100 packets delivered reliably",
		ActualBehavior:   fmt.Sprintf("%d of %d packets received (%.1f%% delivery rate)", packetsReceived, packetsSent, deliveryRate),
		PerformanceMetrics: map[string]interface{}{
			"packets_sent":     packetsSent,
			"packets_received": packetsReceived,
			"delivery_rate":    deliveryRate,
			"packet_loss":      packetsSent - packetsReceived,
		},
		Lessons: []string{
			"UDP doesn't guarantee delivery",
			"Implement acknowledgments for critical data",
			"Monitor packet loss rates",
		},
	}
}

func testUDPOrderingReality(t *testing.T) TestResult {
	// Test UDP packet ordering by sending packets with delays
	
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	
	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	
	var receivedOrder []int
	var mu sync.Mutex
	
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, _, err := server.ReadFromUDP(buffer)
			if err != nil {
				break
			}
			
			var packetNum int
			fmt.Sscanf(string(buffer[:n]), "packet-%d", &packetNum)
			
			mu.Lock()
			receivedOrder = append(receivedOrder, packetNum)
			mu.Unlock()
		}
	}()
	
	client, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	
	// Send packets in order but with varying delays
	sentOrder := []int{1, 2, 3, 4, 5}
	for _, num := range sentOrder {
		data := fmt.Sprintf("packet-%d", num)
		client.Write([]byte(data))
		
		// Add variable delay to potentially cause reordering
		if num == 2 {
			time.Sleep(10 * time.Millisecond)
		} else {
			time.Sleep(1 * time.Millisecond)
		}
	}
	
	time.Sleep(50 * time.Millisecond)
	
	mu.Lock()
	received := make([]int, len(receivedOrder))
	copy(received, receivedOrder)
	mu.Unlock()
	
	inOrder := true
	for i := 1; i < len(received); i++ {
		if received[i] < received[i-1] {
			inOrder = false
			break
		}
	}
	
	return TestResult{
		Success:          true,
		ExpectedBehavior: "Packets received in sent order: [1,2,3,4,5]",
		ActualBehavior:   fmt.Sprintf("Packets received in order: %v (in-order: %v)", received, inOrder),
		PerformanceMetrics: map[string]interface{}{
			"packets_sent":     len(sentOrder),
			"packets_received": len(received),
			"in_order":         inOrder,
		},
		Lessons: []string{
			"UDP doesn't guarantee packet ordering",
			"Implement sequence numbers for ordering",
			"Buffer out-of-order packets when necessary",
		},
	}
}

func testUDPPerformanceReality(t *testing.T) TestResult {
	// Compare raw UDP vs UDP with reliability features
	
	start := time.Now()
	
	// Simple UDP test - measure setup time
	udpConn := NewPracticalUDPConnection(1024, 5*time.Second)
	// Simulate setup without actual connection
	_ = udpConn
	
	setupTime := time.Since(start)
	
	return TestResult{
		Success:          true,
		ExpectedBehavior: "UDP setup should be instantaneous",
		ActualBehavior:   fmt.Sprintf("UDP setup took %v (including error handling)", setupTime),
		PerformanceMetrics: map[string]interface{}{
			"setup_time": setupTime,
		},
		Lessons: []string{
			"Even UDP requires proper setup and error handling",
			"Reliability features add overhead",
			"Balance features vs performance requirements",
		},
	}
}

// Benchmark comparing theory vs practice

func BenchmarkTCPVsUDPTheoryVsPractice(b *testing.B) {
	b.Run("TCPTheoreticalSend", func(b *testing.B) {
		// Theoretical: Just send data
		data := []byte("Hello, World!")
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			// Simulate theoretical instant send
			_ = data
		}
	})
	
	b.Run("TCPPracticalSend", func(b *testing.B) {
		// Practical: Setup connection, handle errors, cleanup
		listener, _ := net.Listen("tcp", "localhost:0")
		defer listener.Close()
		
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go func() {
					buffer := make([]byte, 1024)
					conn.Read(buffer)
					conn.Close()
				}()
			}
		}()
		
		data := []byte("Hello, World!")
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				b.Fatal(err)
			}
			conn.Write(data)
			conn.Close()
		}
	})
}

// Run comprehensive network theory vs practice analysis
func TestComprehensiveNetworkAnalysis(t *testing.T) {
	t.Log("=== Comprehensive Network Programming: Theory vs Practice ===")
	
	// TCP Analysis
	t.Log("\n--- TCP Protocol Analysis ---")
	TestTCPTheoryVsPractice(t)
	
	// UDP Analysis  
	t.Log("\n--- UDP Protocol Analysis ---")
	TestUDPTheoryVsPractice(t)
	
	// Generate final report
	t.Log("\n--- Final Analysis ---")
	t.Log("Key Insights:")
	t.Log("1. Network programming requires extensive error handling beyond protocol specifications")
	t.Log("2. Theoretical protocol behaviors rarely match real-world network conditions")
	t.Log("3. Application-level features often overshadow protocol-level optimizations")
	t.Log("4. Proper testing must account for network variability and failure scenarios")
	t.Log("5. Performance optimization requires understanding both theory and practical limitations")
}