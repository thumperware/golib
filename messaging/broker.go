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

type Event interface {
	Name() string
}

type Message struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type Broker struct {
	urls       string
	connection *nats.Conn
	stream     nats.JetStreamContext
	domain     string
	service    string
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
		domain:  domain,
		service: service,
	}, nil
}

func (b *Broker) WithStream(topics []string) error {
	if len(topics) <= 0 {
		return errors.New("topics is empty")
	}
	domainTopics := []string{}
	for _, t := range topics {
		domainTopics = append(domainTopics, fmt.Sprintf("%s.%s.%s", b.domain, b.service, t))
	}
	if b.connection == nil {
		err := b.Connect()
		if err != nil {
			return err
		}
	}
	js, err := b.connection.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return err
	}
	b.stream = js
	_, err = b.stream.AddStream(&nats.StreamConfig{
		Name:     fmt.Sprintf("%s-%s", b.domain, b.service),
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
	b.connection = nc
	return nil
}

func (b *Broker) Disconnect() error {
	return b.connection.Drain()
}

func (b *Broker) Publish(topic string, data Event) error {
	if topic == "" {
		return errors.New("publish topic is empty")
	}
	if data == nil {
		return errors.New("publish data is nil")
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg := &Message{
		Name: data.Name(),
		Data: dataBytes,
	}
	msgJson, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return b.connection.Publish(fmt.Sprintf("%s.%s.%s", b.domain, b.service, topic), msgJson)
}

func (b *Broker) PublishStream(topic string, data Event) error {
	if topic == "" {
		return errors.New("publish stream topic is empty")
	}
	if data == nil {
		return errors.New("publish stream data is nil")
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg := &Message{
		Name: data.Name(),
		Data: dataBytes,
	}
	msgJson, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = b.stream.Publish(fmt.Sprintf("%s.%s.%s", b.domain, b.service, topic), msgJson)
	if err != nil {
		return err
	}
	return nil
}

type Subscriber struct {
	subscriberName string
	broker         *Broker
}

func NewSubscriber(broker *Broker) *Subscriber {
	return &Subscriber{
		subscriberName: fmt.Sprintf("%s-%s", broker.domain, broker.service),
		broker:         broker,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, domain string, service string, topic string, handler func(ctx context.Context, msg Message) error) error {
	msgs := make(chan *nats.Msg)
	subject := fmt.Sprintf("%s.%s.%s", domain, service, topic)
	queueName := fmt.Sprintf("%s-%s", s.subscriberName, strings.ReplaceAll(subject, ".", "-"))
	sub, err := s.broker.connection.QueueSubscribeSyncWithChan(subject, queueName, msgs)
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
				var data Message
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

func (s *Subscriber) SubscribeStream(ctx context.Context, domain string, service string, topic string, handler func(ctx context.Context, msg Message) error) error {
	subject := fmt.Sprintf("%s.%s.%s", domain, service, topic)
	queueName := fmt.Sprintf("%s-%s", s.subscriberName, strings.ReplaceAll(subject, ".", "-"))
	sub, err := s.broker.stream.PullSubscribe(subject, queueName, nats.PullMaxWaiting(128))
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
					var data Message
					err := json.Unmarshal(msg.Data, &data)
					if err != nil {
						logging.TraceLogger(ctx).
							Err(err).
							Msgf("failed to unmarshal stream message with subject %s", msg.Subject)
						err := msg.Nak()
						if err != nil {
							logging.TraceLogger(ctx).
								Err(err).
								Msgf("ack error for subject %s", msg.Subject)
						}
						continue
					}
					err = handler(ctx, data)
					if err != nil {
						logging.TraceLogger(ctx).
							Err(err).
							Msgf("stream handler error for subject %s", msg.Subject)
						err := msg.Nak()
						if err != nil {
							logging.TraceLogger(ctx).
								Err(err).
								Msgf("ack error for subject %s", msg.Subject)
						}
					} else {
						err := msg.Ack()
						if err != nil {
							logging.TraceLogger(ctx).
								Err(err).
								Msgf("ack error for subject %s", msg.Subject)
						}
					}
				}
			}
		}
	}()
	return nil
}
