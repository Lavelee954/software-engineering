package messagebus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type NATSBus struct {
	conn         *nats.Conn
	subscriptions map[string]*nats.Subscription
	mu           sync.RWMutex
	logger       interfaces.Logger
	metrics      interfaces.MetricsCollector
}

type Config struct {
	URL              string
	MaxReconnects    int
	ReconnectWait    time.Duration
	ConnectionTimeout time.Duration
	DrainTimeout     time.Duration
}

func NewNATSBus(config Config, logger interfaces.Logger, metrics interfaces.MetricsCollector) (*NATSBus, error) {
	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.Timeout(config.ConnectionTimeout),
		nats.DrainTimeout(config.DrainTimeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", interfaces.Field{Key: "error", Value: err})
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", interfaces.Field{Key: "url", Value: nc.ConnectedUrl()})
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSBus{
		conn:          conn,
		subscriptions: make(map[string]*nats.Subscription),
		logger:        logger,
		metrics:       metrics,
	}, nil
}

func (nb *NATSBus) Publish(ctx context.Context, topic string, message interface{}) error {
	start := time.Now()
	defer func() {
		nb.metrics.RecordDuration("message_bus_publish_duration", time.Since(start).Seconds(), map[string]string{
			"topic": topic,
		})
	}()

	data, err := json.Marshal(message)
	if err != nil {
		nb.metrics.IncrementCounter("message_bus_publish_errors", map[string]string{
			"topic": topic,
			"error": "marshal_failed",
		})
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := nb.conn.Publish(topic, data); err != nil {
		nb.metrics.IncrementCounter("message_bus_publish_errors", map[string]string{
			"topic": topic,
			"error": "publish_failed",
		})
		nb.logger.Error("Failed to publish message",
			interfaces.Field{Key: "topic", Value: topic},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	nb.metrics.IncrementCounter("message_bus_published", map[string]string{
		"topic": topic,
	})

	nb.logger.Debug("Message published",
		interfaces.Field{Key: "topic", Value: topic},
		interfaces.Field{Key: "size", Value: len(data)},
	)

	return nil
}

func (nb *NATSBus) Subscribe(ctx context.Context, topic string, handler interfaces.MessageHandler) error {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if _, exists := nb.subscriptions[topic]; exists {
		return fmt.Errorf("already subscribed to topic: %s", topic)
	}

	msgHandler := func(msg *nats.Msg) {
		start := time.Now()
		defer func() {
			nb.metrics.RecordDuration("message_bus_handle_duration", time.Since(start).Seconds(), map[string]string{
				"topic": topic,
			})
		}()

		ctx := context.Background()
		if err := handler(ctx, msg.Data); err != nil {
			nb.metrics.IncrementCounter("message_bus_handle_errors", map[string]string{
				"topic": topic,
			})
			nb.logger.Error("Message handler failed",
				interfaces.Field{Key: "topic", Value: topic},
				interfaces.Field{Key: "error", Value: err},
			)
			return
		}

		nb.metrics.IncrementCounter("message_bus_handled", map[string]string{
			"topic": topic,
		})
	}

	sub, err := nb.conn.Subscribe(topic, msgHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	nb.subscriptions[topic] = sub

	nb.logger.Info("Subscribed to topic",
		interfaces.Field{Key: "topic", Value: topic},
	)

	return nil
}

func (nb *NATSBus) Unsubscribe(topic string) error {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	sub, exists := nb.subscriptions[topic]
	if !exists {
		return fmt.Errorf("not subscribed to topic: %s", topic)
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, err)
	}

	delete(nb.subscriptions, topic)

	nb.logger.Info("Unsubscribed from topic",
		interfaces.Field{Key: "topic", Value: topic},
	)

	return nil
}

func (nb *NATSBus) Close() error {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	for topic, sub := range nb.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			nb.logger.Warn("Failed to unsubscribe",
				interfaces.Field{Key: "topic", Value: topic},
				interfaces.Field{Key: "error", Value: err},
			)
		}
	}

	nb.subscriptions = make(map[string]*nats.Subscription)

	if nb.conn != nil {
		nb.conn.Close()
	}

	nb.logger.Info("NATS message bus closed")
	return nil
}

func (nb *NATSBus) IsConnected() bool {
	return nb.conn != nil && nb.conn.IsConnected()
}

func (nb *NATSBus) Stats() nats.Statistics {
	if nb.conn == nil {
		return nats.Statistics{}
	}
	return nb.conn.Stats()
}