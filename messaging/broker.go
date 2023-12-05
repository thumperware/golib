package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/logging"
)

type Broker struct {
	urls       string
	Connection *nats.Conn
	Stream     nats.JetStreamContext
	Domain     string
	Service    string
}

func NewBroker(cfg config.CfgManager, domain string, service string) (*Broker, error) {
	if domain == "" {
		return nil, errors.New("domain is empty")
	}
	if service == "" {
		return nil, errors.New("service is empty")
	}
	urls, err := cfg.GetValue(context.Background(), "NATS_URLS")
	if err != nil {
		return nil, err
	}
	return &Broker{
		urls:    urls,
		Domain:  domain,
		Service: service,
	}, nil
}

func (b *Broker) WithStream(topics []string) error {
	if len(topics) <= 0 {
		return errors.New("topics is empty")
	}
	domainTopics := []string{}
	for _, t := range topics {
		domainTopics = append(domainTopics, fmt.Sprintf("%s.%s.%s", b.Domain, b.Service, t))
	}
	if b.Connection == nil {
		err := b.Connect()
		if err != nil {
			return err
		}
	}
	js, err := b.Connection.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return err
	}
	b.Stream = js
	_, err = b.Stream.AddStream(&nats.StreamConfig{
		Name:     fmt.Sprintf("%s-%s", b.Domain, b.Service),
		Subjects: domainTopics,
	})
	if err != nil {
		return err
	}
	return nil
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

func (b *Broker) PublishStream(topic string, data any) error {
	dataJson, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = b.Stream.Publish(fmt.Sprintf("%s.%s.%s", b.Domain, b.Service, topic), dataJson)
	if err != nil {
		return err
	}
	return nil
}

type Subscriber[T any] struct {
	subscriberName string
	broker         *Broker
}

func NewSubscriber[T any](broker *Broker) *Subscriber[T] {
	return &Subscriber[T]{
		subscriberName: fmt.Sprintf("%s-%s", broker.Domain, broker.Service),
		broker:         broker,
	}
}

func (s *Subscriber[T]) Subscribe(ctx context.Context, domain string, service string, topic string, handler func(ctx context.Context, data T) error) error {
	msgs := make(chan *nats.Msg)
	subject := fmt.Sprintf("%s.%s.%s", domain, service, topic)
	queueName := fmt.Sprintf("%s-%s", s.subscriberName, strings.ReplaceAll(subject, ".", "-"))
	sub, err := s.broker.Connection.QueueSubscribeSyncWithChan(subject, queueName, msgs)
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
				return
			case msg := <-msgs:
				var data T
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					logging.TraceLogger(ctx).
						Err(err).
						Msgf("failed to unmarshal message with subject %s", msg.Subject)
					continue
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

func (s *Subscriber[T]) SubscribeStream(ctx context.Context, domain string, service string, topic string, handler func(ctx context.Context, data T) error) error {
	subject := fmt.Sprintf("%s.%s.%s", domain, service, topic)
	queueName := fmt.Sprintf("%s-%s", s.subscriberName, strings.ReplaceAll(subject, ".", "-"))
	sub, err := s.broker.Stream.PullSubscribe(subject, queueName, nats.PullMaxWaiting(128))
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
						Msgf("failed to unsubscribe from subject stream %s", subject)
				}
				return
			default:
				msgs, _ := sub.Fetch(10, nats.Context(ctx))
				for _, msg := range msgs {
					var data T
					err := json.Unmarshal(msg.Data, &data)
					if err != nil {
						logging.TraceLogger(ctx).
							Err(err).
							Msgf("failed to unmarshal stream message with subject %s", msg.Subject)
						msg.Nak()
						continue
					}
					err = handler(ctx, data)
					if err != nil {
						logging.TraceLogger(ctx).
							Err(err).
							Msgf("stream handler error for subject %s", msg.Subject)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()
	return nil
}
