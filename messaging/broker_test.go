package messaging_test

import (
	"context"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thumperq/golib/messaging"
)

type orderCreated struct {
	Name      string `json:"name"`
	OrderId   string `json:"orderId"`
	OrderType string `json:"orderType"`
}

type mockCfgManager struct {
	Value string
}

func (m *mockCfgManager) GetValue(ctx context.Context, key string) (string, error) {
	return m.Value, nil
}

func (m *mockCfgManager) GetValueOfDomainService(ctx context.Context, domain string, service string, key string) (string, error) {
	return m.Value, nil
}
func startNatsServer() *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = 4222
	srv := natsserver.RunServer(&opts)
	err := srv.EnableJetStream(&server.JetStreamConfig{})
	if err != nil {
		panic(err)
	}
	return srv
}

func TestPublishAndSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ns := startNatsServer()
	mockCfgManager := &mockCfgManager{Value: ns.ClientURL()}
	broker, err := messaging.NewBroker(mockCfgManager, "wms", "ordering")
	require.NoError(t, err)
	err = broker.Connect()
	require.NoError(t, err)
	subscriber := messaging.NewSubscriber[orderCreated](broker)
	subscriber.Subscribe(ctx, "wms", "ordering", "order", func(ctx context.Context, data orderCreated) error {
		require.Equal(t, "123", data.OrderId)
		require.Equal(t, "normal", data.OrderType)
		cancel()
		broker.Disconnect()
		ns.Shutdown()
		return nil
	})
	broker.Publish("order", &orderCreated{
		Name:      "orderCreated",
		OrderId:   "123",
		OrderType: "normal",
	})
	ns.WaitForShutdown()
}

func TestStreamPublishAndSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ns := startNatsServer()
	mockCfgManager := &mockCfgManager{Value: ns.ClientURL()}
	broker, err := messaging.NewBroker(mockCfgManager, "wms", "ordering")
	require.NoError(t, err)
	err = broker.WithStream([]string{"order"})
	require.NoError(t, err)
	subscriber := messaging.NewSubscriber[orderCreated](broker)
	subscriber.SubscribeStream(ctx, "wms", "ordering", "order", func(ctx context.Context, data orderCreated) error {
		require.Equal(t, "orderCreated", data.Name)
		require.Equal(t, "123", data.OrderId)
		require.Equal(t, "normal", data.OrderType)
		broker.Disconnect()
		ns.Shutdown()
		return nil
	})
	broker.PublishStream("order", &orderCreated{
		Name:      "orderCreated",
		OrderId:   "123",
		OrderType: "normal",
	})
	ns.WaitForShutdown()
}
