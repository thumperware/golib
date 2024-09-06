package messaging

import "context"

type Consumer interface {
	Handle(ctx context.Context, msg Message) error
}

type Worker interface {
	Run(consumer Consumer) func(ctx context.Context, domain string, service string, topic string) error
}

type worker struct {
	broker *Broker
}

func NewWorker(broker *Broker) Worker {
	return &worker{
		broker: broker,
	}
}

func (cw *worker) Run(consumer Consumer) func(ctx context.Context, domain string, service string, topic string) error {
	return func(ctx context.Context, domain string, service string, topic string) error {
		return NewSubscriber(cw.broker).
			Subscribe(ctx, domain, service, topic, consumer.Handle)
	}
}
