package messagebus

import (
	"context"
	"sync"

	ifs "github.com/system-trading/core/internal/usecases/interfaces"
)

// MockMessageBus is a mock implementation of MessageBus for testing
type MockMessageBus struct {
	handlers map[string]ifs.MessageHandler
	messages []MockMessage
	mu       sync.RWMutex
}

// MockMessage represents a published message
type MockMessage struct {
	Topic   string
	Message interface{}
}

// NewMockMessageBus creates a new mock message bus
func NewMockMessageBus() *MockMessageBus {
	return &MockMessageBus{
		handlers: make(map[string]ifs.MessageHandler),
		messages: []MockMessage{},
	}
}

// Publish stores the message for testing purposes
func (m *MockMessageBus) Publish(ctx context.Context, topic string, message interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.messages = append(m.messages, MockMessage{
		Topic:   topic,
		Message: message,
	})
	return nil
}

// Subscribe registers a handler for a topic
func (m *MockMessageBus) Subscribe(ctx context.Context, topic string, handler ifs.MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.handlers[topic] = handler
	return nil
}

// Close cleans up the mock bus
func (m *MockMessageBus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.handlers = make(map[string]ifs.MessageHandler)
	m.messages = []MockMessage{}
	return nil
}

// GetHandler returns the handler for a topic (testing helper)
func (m *MockMessageBus) GetHandler(topic string) ifs.MessageHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.handlers[topic]
}

// GetMessages returns all published messages (testing helper)
func (m *MockMessageBus) GetMessages() []MockMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	messages := make([]MockMessage, len(m.messages))
	copy(messages, m.messages)
	return messages
}

// GetMessagesByTopic returns messages for a specific topic (testing helper)
func (m *MockMessageBus) GetMessagesByTopic(topic string) []MockMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var filtered []MockMessage
	for _, msg := range m.messages {
		if msg.Topic == topic {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// ClearMessages clears all stored messages (testing helper)
func (m *MockMessageBus) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.messages = []MockMessage{}
}

// IsConnected always returns true for mock (testing helper)
func (m *MockMessageBus) IsConnected() bool {
	return true
}