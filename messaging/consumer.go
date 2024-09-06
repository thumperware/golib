package messaging

import "context"

type Consumer interface {
	Handle(ctx context.Context, msg Message) error
}

type ConsumerWorker interface {
	Consume(consumer Consumer) func(ctx context.Context, domain string, service string, topic string) error
}

type consumerWorker struct {
	broker *Broker
}

func NewConsumerWorker(broker *Broker) ConsumerWorker {
	return &consumerWorker{
		broker: broker,
	}
}

func (cw *consumerWorker) Consume(consumer Consumer) func(ctx context.Context, domain string, service string, topic string) error {
	return func(ctx context.Context, domain string, service string, topic string) error {
		return NewSubscriber(cw.broker).
			Subscribe(ctx, domain, service, topic, consumer.Handle)
	}
}
