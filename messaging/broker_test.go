package messaging_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"

	"github.com/stretchr/testify/require"
	"github.com/thumperq/golib/messaging"
)

type orderCreated struct {
	EventName string `json:"name"`
	OrderId   string `json:"orderId"`
	OrderType string `json:"orderType"`
}

func (o orderCreated) Name() string {
	return o.EventName
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
	subscriber := messaging.NewSubscriber(broker)
	err = subscriber.Subscribe(ctx, "wms", "ordering", "order", func(ctx context.Context, msg messaging.Message) error {
		require.Equal(t, "orderCreated", msg.Name)
		var event orderCreated
		err := json.Unmarshal(msg.Data, &event)
		require.NoError(t, err)
		require.Equal(t, "123", event.OrderId)
		require.Equal(t, "normal", event.OrderType)
		cancel()
		err = broker.Disconnect()
		require.NoError(t, err)
		ns.Shutdown()
		return nil
	})
	require.NoError(t, err)
	err = broker.Publish("order", &orderCreated{
		EventName: "orderCreated",
		OrderId:   "123",
		OrderType: "normal",
	})
	require.NoError(t, err)
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
	subscriber := messaging.NewSubscriber(broker)
	err = subscriber.SubscribeStream(ctx, "wms", "ordering", "order", func(ctx context.Context, msg messaging.Message) error {
		require.Equal(t, "orderCreated", msg.Name)
		var event orderCreated
		err := json.Unmarshal(msg.Data, &event)
		require.NoError(t, err)
		require.Equal(t, "123", event.OrderId)
		require.Equal(t, "normal", event.OrderType)
		err = broker.Disconnect()
		require.NoError(t, err)
		ns.Shutdown()
		return nil
	})
	require.NoError(t, err)
	err = broker.PublishStream("order", &orderCreated{
		EventName: "orderCreated",
		OrderId:   "123",
		OrderType: "normal",
	})
	require.NoError(t, err)
	ns.WaitForShutdown()
}
