package application

import (
	"reflect"

	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/messaging"
)

var Factory AppFactory

type AppFactory interface {
	Register(newApp func(AppFactory) any) AppFactory
	Broker() *messaging.Broker
	Config() config.CfgManager
	Get(typ reflect.Type) any
}

type ApplicationFactory struct {
	broker *messaging.Broker
	cfg    config.CfgManager
	apps   map[reflect.Type]any
}

func NewApplicationFactory(broker *messaging.Broker, cfg config.CfgManager) AppFactory {
	return &ApplicationFactory{
		broker: broker,
		cfg:    cfg,
	}
}

func (af *ApplicationFactory) Register(newApp func(AppFactory) any) AppFactory {
	app := newApp(af)
	af.apps[reflect.TypeOf(app)] = app
	return af
}

func (af *ApplicationFactory) Broker() *messaging.Broker {
	return af.broker
}

func (af *ApplicationFactory) Config() config.CfgManager {
	return af.cfg
}

func (af *ApplicationFactory) Get(typ reflect.Type) any {
	return af.apps[typ]
}

func GetApp[T any]() T {
	return Factory.Get(reflect.TypeFor[T]()).(T)
}
