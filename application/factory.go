package application

import (
	"reflect"

	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/messaging"
)

type AppFactory interface {
	Register(newApp func(AppFactory) any) AppFactory
	Broker() *messaging.Broker
	Config() config.CfgManager
	Get(typ reflect.Type) any
}

type appFactory struct {
	broker *messaging.Broker
	cfg    config.CfgManager
	apps   map[reflect.Type]any
}

func NewApplicationFactory(broker *messaging.Broker, cfg config.CfgManager) AppFactory {
	return &appFactory{
		broker: broker,
		cfg:    cfg,
	}
}

func (af *appFactory) Register(newApp func(AppFactory) any) AppFactory {
	app := newApp(af)
	af.apps[reflect.TypeOf(app)] = app
	return af
}

func (af *appFactory) Broker() *messaging.Broker {
	return af.broker
}

func (af *appFactory) Config() config.CfgManager {
	return af.cfg
}

func (af *appFactory) Get(typ reflect.Type) any {
	return af.apps[typ]
}
