package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/thumperq/golib/logging"
)

type Broker struct {
	urls       string
	Connection *nats.Conn
	Domain     string
	Service    string
}

func NewBroker(urls string, domain string, service string) (*Broker, error) {
	if urls == "" {
		return nil, errors.New("urls is empty")
	}
	if domain == "" {
		return nil, errors.New("domain is empty")
	}
	if service == "" {
		return nil, errors.New("service is empty")
	}
	return &Broker{
		urls:    urls,
		Domain:  domain,
		Service: service,
	}, nil
}

func (b *Broker) Connect() error {
	nc, err := nats.Connect(b.urls, nats.MaxReconnects(10), nats.ReconnectWait(time.Second))
	if err != nil {
		return err
	}
	b.Connection = nc
	return nil
}

func (b *Broker) Disconnect() error {
	return b.Connection.Drain()
}

func (b *Broker) Publish(topic string, data any) error {
	if topic == "" {
		return errors.New("publish topic is empty")
	}
	if data == nil {
		return errors.New("publish data is nil")
	}
	dataJson, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.Connection.Publish(fmt.Sprintf("%s.%s.%s", b.Domain, b.Service, topic), dataJson)
}

type Subscriber[T any] struct {
	queueName string
	broker    *Broker
}

func NewSubscriber[T any](broker *Broker) *Subscriber[T] {
	queueName := fmt.Sprintf("%s.%s", broker.Domain, broker.Service)
	return &Subscriber[T]{
		queueName: queueName,
		broker:    broker,
	}
}

func (s *Subscriber[T]) SubscribeWithSubject(ctx context.Context, subject string, handler func(ctx context.Context, data T) error) error {
	msgs := make(chan *nats.Msg)
	sub, err := s.broker.Connection.QueueSubscribeSyncWithChan(subject, s.queueName, msgs)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				err := sub.Unsubscribe()
				if err != nil {
					logging.TraceLogger(ctx).
						Err(err).
						Msgf("failed to unsubscribe from subject %s", subject)
				}
			case msg := <-msgs:
				var data T
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					logging.TraceLogger(ctx).
						Err(err).
						Msgf("failed to unmarshal message with subject %s", msg.Subject)
				}
				err = handler(ctx, data)
				if err != nil {
					logging.TraceLogger(ctx).
						Err(err).
						Msgf("handler error for subject %s", msg.Subject)
				}
			}
		}
	}()
	return nil
}

func (s *Subscriber[T]) Subscribe(ctx context.Context, domain string, service string, topic string, handler func(ctx context.Context, data T) error) error {
	return s.SubscribeWithSubject(ctx, fmt.Sprintf("%s.%s.%s", domain, service, topic), handler)
}
