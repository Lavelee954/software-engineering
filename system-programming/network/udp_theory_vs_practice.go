package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// TheoreticalUDPConnection represents ideal UDP behavior
type TheoreticalUDPConnection struct {
	connectionless bool
	unreliable     bool
	fast           bool
	lightweight    bool
}

// PracticalUDPConnection represents real UDP implementation challenges
type PracticalUDPConnection struct {
	conn                net.PacketConn
	remoteAddr          net.Addr
	localAddr           net.Addr
	packetsSent         int64
	packetsReceived     int64
	packetsLost         int64
	duplicatePackets    int64
	outOfOrderPackets   int64
	maxPacketSize       int
	readTimeout         time.Duration
	writeTimeout        time.Duration
	sequenceNumber      uint32
	expectedSequence    uint32
	mu                  sync.RWMutex
	packetBuffer        map[uint32][]byte // Buffer for out-of-order packets
}

// UDPPacket represents a practical UDP packet with metadata
type UDPPacket struct {
	SequenceNumber uint32
	Timestamp      time.Time
	Data           []byte
	Checksum       uint32
	Size           int
}

// UDPStatistics tracks real-world UDP behavior
type UDPStatistics struct {
	PacketsSent        int64
	PacketsReceived    int64
	PacketsLost        int64
	DuplicatePackets   int64
	OutOfOrderPackets  int64
	AverageRTT         time.Duration
	PacketLossRate     float64
	JitterVariance     time.Duration
}

func NewPracticalUDPConnection(maxPacketSize int, timeout time.Duration) *PracticalUDPConnection {
	return &PracticalUDPConnection{
		maxPacketSize:    maxPacketSize,
		readTimeout:      timeout,
		writeTimeout:     timeout,
		packetBuffer:     make(map[uint32][]byte),
		expectedSequence: 1,
		sequenceNumber:   1,
	}
}

// Listen demonstrates UDP server setup complexities
func (p *PracticalUDPConnection) Listen(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Theory: UDP listening is simple - just bind to port
	// Practice: Handle address conflicts, permission issues, multiple interfaces
	
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port: %w", err)
	}
	
	p.conn = conn
	p.localAddr = conn.LocalAddr()
	
	log.Printf("UDP server listening on %s", p.localAddr)
	return nil
}

// Connect demonstrates UDP "connection" establishment
func (p *PracticalUDPConnection) Connect(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Theory: UDP is connectionless, no connection needed
	// Practice: We still need to resolve addresses and set up local socket
	
	remoteAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("failed to resolve remote address: %w", err)
	}
	
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	
	p.conn = conn
	p.remoteAddr = remoteAddr
	p.localAddr = conn.LocalAddr()
	
	return nil
}

// SendPacket demonstrates UDP sending challenges
func (p *PracticalUDPConnection) SendPacket(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Theory: UDP sends packets immediately, fire-and-forget
	// Practice: Handle MTU limits, fragmentation, network errors
	
	if len(data) > p.maxPacketSize {
		return fmt.Errorf("packet size %d exceeds maximum %d", len(data), p.maxPacketSize)
	}
	
	// Create packet with sequence number for tracking
	packet := UDPPacket{
		SequenceNumber: p.sequenceNumber,
		Timestamp:      time.Now(),
		Data:           data,
		Size:           len(data),
	}
	
	// Calculate simple checksum
	packet.Checksum = calculateChecksum(data)
	
	// Serialize packet
	serialized := serializePacket(packet)
	
	// Set write deadline
	if udpConn, ok := p.conn.(*net.UDPConn); ok {
		udpConn.SetWriteDeadline(time.Now().Add(p.writeTimeout))
	}
	
	// Practical consideration: Handle partial writes and errors
	bytesSent, err := p.conn.WriteTo(serialized, p.remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send UDP packet: %w", err)
	}
	
	if bytesSent != len(serialized) {
		return fmt.Errorf("partial packet sent: %d of %d bytes", bytesSent, len(serialized))
	}
	
	p.packetsSent++
	p.sequenceNumber++
	
	log.Printf("Sent UDP packet #%d (%d bytes)", packet.SequenceNumber, len(data))
	return nil
}

// ReceivePacket demonstrates UDP receiving complexities
func (p *PracticalUDPConnection) ReceivePacket() (*UDPPacket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Theory: UDP packets arrive in order, one at a time
	// Practice: Packets can arrive out of order, be duplicated, or lost
	
	buffer := make([]byte, p.maxPacketSize)
	
	// Set read deadline
	if udpConn, ok := p.conn.(*net.UDPConn); ok {
		udpConn.SetReadDeadline(time.Now().Add(p.readTimeout))
	}
	
	bytesRead, addr, err := p.conn.ReadFrom(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("receive timeout: %w", err)
		}
		return nil, fmt.Errorf("failed to receive UDP packet: %w", err)
	}
	
	// Practical consideration: Verify sender address
	if p.remoteAddr != nil && addr.String() != p.remoteAddr.String() {
		log.Printf("Received packet from unexpected address: %s", addr)
	}
	
	// Deserialize packet
	packet, err := deserializePacket(buffer[:bytesRead])
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize packet: %w", err)
	}
	
	// Practical consideration: Verify checksum
	expectedChecksum := calculateChecksum(packet.Data)
	if packet.Checksum != expectedChecksum {
		return nil, fmt.Errorf("packet checksum mismatch: expected %d, got %d", 
			expectedChecksum, packet.Checksum)
	}
	
	p.packetsReceived++
	
	// Practical consideration: Handle out-of-order packets
	if packet.SequenceNumber != p.expectedSequence {
		if packet.SequenceNumber < p.expectedSequence {
			// Duplicate packet
			p.duplicatePackets++
			log.Printf("Received duplicate packet #%d", packet.SequenceNumber)
			return nil, fmt.Errorf("duplicate packet received")
		} else {
			// Out of order packet - buffer it
			p.outOfOrderPackets++
			p.packetBuffer[packet.SequenceNumber] = packet.Data
			log.Printf("Received out-of-order packet #%d (expected #%d)", 
				packet.SequenceNumber, p.expectedSequence)
			
			// Check if we can deliver buffered packets
			return p.checkBufferedPackets()
		}
	}
	
	p.expectedSequence++
	
	log.Printf("Received UDP packet #%d (%d bytes)", packet.SequenceNumber, len(packet.Data))
	return packet, nil
}

// checkBufferedPackets tries to deliver consecutive buffered packets
func (p *PracticalUDPConnection) checkBufferedPackets() (*UDPPacket, error) {
	if data, exists := p.packetBuffer[p.expectedSequence]; exists {
		delete(p.packetBuffer, p.expectedSequence)
		
		packet := &UDPPacket{
			SequenceNumber: p.expectedSequence,
			Data:           data,
			Size:           len(data),
		}
		
		p.expectedSequence++
		log.Printf("Delivered buffered packet #%d", packet.SequenceNumber)
		return packet, nil
	}
	
	return nil, fmt.Errorf("waiting for packet #%d", p.expectedSequence)
}

// GetStatistics returns detailed UDP connection statistics
func (p *PracticalUDPConnection) GetStatistics() UDPStatistics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var packetLossRate float64
	if p.packetsSent > 0 {
		p.packetsLost = p.packetsSent - p.packetsReceived
		packetLossRate = float64(p.packetsLost) / float64(p.packetsSent) * 100
	}
	
	return UDPStatistics{
		PacketsSent:       p.packetsSent,
		PacketsReceived:   p.packetsReceived,
		PacketsLost:       p.packetsLost,
		DuplicatePackets:  p.duplicatePackets,
		OutOfOrderPackets: p.outOfOrderPackets,
		PacketLossRate:    packetLossRate,
	}
}

// Close cleans up UDP connection
func (p *PracticalUDPConnection) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// Helper functions for packet serialization
func serializePacket(packet UDPPacket) []byte {
	// Simple serialization - in practice, use protobuf or similar
	result := make([]byte, 0, len(packet.Data)+16)
	
	// Add sequence number (4 bytes)
	result = append(result, byte(packet.SequenceNumber>>24), 
		byte(packet.SequenceNumber>>16), 
		byte(packet.SequenceNumber>>8), 
		byte(packet.SequenceNumber))
	
	// Add checksum (4 bytes)
	result = append(result, byte(packet.Checksum>>24),
		byte(packet.Checksum>>16),
		byte(packet.Checksum>>8),
		byte(packet.Checksum))
	
	// Add timestamp (8 bytes)
	timestamp := packet.Timestamp.Unix()
	result = append(result, byte(timestamp>>56),
		byte(timestamp>>48),
		byte(timestamp>>40),
		byte(timestamp>>32),
		byte(timestamp>>24),
		byte(timestamp>>16),
		byte(timestamp>>8),
		byte(timestamp))
	
	// Add data
	result = append(result, packet.Data...)
	
	return result
}

func deserializePacket(data []byte) (*UDPPacket, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("packet too small: %d bytes", len(data))
	}
	
	packet := &UDPPacket{}
	
	// Extract sequence number
	packet.SequenceNumber = uint32(data[0])<<24 | uint32(data[1])<<16 | 
		uint32(data[2])<<8 | uint32(data[3])
	
	// Extract checksum
	packet.Checksum = uint32(data[4])<<24 | uint32(data[5])<<16 | 
		uint32(data[6])<<8 | uint32(data[7])
	
	// Extract timestamp
	timestamp := int64(data[8])<<56 | int64(data[9])<<48 | 
		int64(data[10])<<40 | int64(data[11])<<32 |
		int64(data[12])<<24 | int64(data[13])<<16 | 
		int64(data[14])<<8 | int64(data[15])
	packet.Timestamp = time.Unix(timestamp, 0)
	
	// Extract data
	packet.Data = data[16:]
	packet.Size = len(packet.Data)
	
	return packet, nil
}

func calculateChecksum(data []byte) uint32 {
	// Simple checksum - in practice, use CRC32 or similar
	var checksum uint32
	for _, b := range data {
		checksum += uint32(b)
	}
	return checksum
}

// GenerateUDPReflections provides insights into UDP theory vs practice
func GenerateUDPReflections() []TheoryReflection {
	return []TheoryReflection{
		{
			ExpectedBehavior: "UDP is connectionless, so no setup required",
			ActualBehavior:   "Still need address resolution, socket binding, and error handling",
			Lesson:           "Even connectionless protocols require careful setup and configuration",
		},
		{
			ExpectedBehavior: "UDP packets are delivered immediately and independently",
			ActualBehavior:   "Packets can be lost, duplicated, reordered, or delayed",
			Lesson:           "Implement sequence numbers, duplicate detection, and reordering logic",
		},
		{
			ExpectedBehavior: "UDP has no flow control, so sending is always fast",
			ActualBehavior:   "Network congestion can still cause delays and packet loss",
			Lesson:           "Monitor packet loss rates and implement application-level flow control",
		},
		{
			ExpectedBehavior: "UDP packets preserve message boundaries automatically",
			ActualBehavior:   "Large packets may be fragmented at IP level, causing complexity",
			Lesson:           "Stay within MTU limits and handle fragmentation scenarios",
		},
		{
			ExpectedBehavior: "UDP is unreliable, so expect some packet loss",
			ActualBehavior:   "Packet loss patterns vary greatly based on network conditions",
			Lesson:           "Implement adaptive strategies based on measured packet loss rates",
		},
		{
			ExpectedBehavior: "UDP has minimal overhead compared to TCP",
			ActualBehavior:   "Application-level reliability features can add significant overhead",
			Lesson:           "Balance reliability features with performance requirements",
		},
	}
}

// DemonstrateUDPRealities shows practical UDP programming challenges
func DemonstrateUDPRealities() {
	fmt.Println("=== UDP Theory vs Practice Demonstration ===")
	fmt.Println()
	
	reflections := GenerateUDPReflections()
	for i, reflection := range reflections {
		fmt.Printf("Lesson %d:\n", i+1)
		fmt.Printf("Theory: %s\n", reflection.ExpectedBehavior)
		fmt.Printf("Practice: %s\n", reflection.ActualBehavior)
		fmt.Printf("Lesson: %s\n\n", reflection.Lesson)
	}
}

func DemonstrateUDPRealitiesMain() {
	DemonstrateUDPRealities()
}