package messaging_test

import (
	"context"
	"log"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
	"github.com/thumperq/golib/messaging"
)

type order struct {
	OrderId   string `json:"orderId"`
	OrderType string `json:"orderType"`
}

func TestPublishAndSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ns := startNatsServer()
	broker, err := messaging.NewBroker(ns.ClientURL(), "wms", "ordering")
	require.NoError(t, err)
	err = broker.Connect()
	require.NoError(t, err)
	subscriber := messaging.NewSubscriber[order](broker)
	subscriber.Subscribe(ctx, "wms", "ordering", "orderCreated", func(ctx context.Context, data order) error {
		require.Equal(t, "123", data.OrderId)
		require.Equal(t, "normal", data.OrderType)
		broker.Disconnect()
		ns.Shutdown()
		return nil
	})
	broker.Publish("orderCreated", &order{
		OrderId:   "123",
		OrderType: "normal",
	})
	ns.WaitForShutdown()
}

func startNatsServer() *server.Server {
	opts := &server.Options{
		ServerName:     "local_nats_server",
		Host:           "localhost",
		Port:           15000,
		NoLog:          false,
		NoSigs:         false,
		MaxControlLine: 4096,
		MaxPayload:     65536,
	}
	srv, err := server.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}
	err = server.Run(srv)
	if err != nil {
		log.Fatal("Failed to start NATS server:", err)
	}
	return srv
}
