package application

import (
	"reflect"

	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/database"
	"github.com/thumperq/golib/messaging"
)

var Factory AppFactory

type AppFactory interface {
	Register(newApp func() any) AppFactory
	WithDbFactory(provide func(dbFactory database.DbFactory)) AppFactory
	WithBroker(provide func(broker *messaging.Broker)) AppFactory
	WithConfig(provide func(cfg config.CfgManager)) AppFactory
	Get(typ reflect.Type) any
}

type ApplicationFactory struct {
	dbFactory database.DbFactory
	broker    *messaging.Broker
	cfg       config.CfgManager
	apps      map[reflect.Type]any
}

func NewApplicationFactory(dbFactory database.DbFactory, broker *messaging.Broker, cfg config.CfgManager) AppFactory {
	return &ApplicationFactory{
		dbFactory: dbFactory,
		broker:    broker,
		cfg:       cfg,
	}
}

func (af *ApplicationFactory) Register(newApp func() any) AppFactory {
	app := newApp()
	af.apps[reflect.TypeOf(app)] = app
	return af
}

func (af *ApplicationFactory) WithDbFactory(provide func(dbFactory database.DbFactory)) AppFactory {
	provide(af.dbFactory)
	return af
}

func (af *ApplicationFactory) WithBroker(provide func(broker *messaging.Broker)) AppFactory {
	provide(af.broker)
	return af
}

func (af *ApplicationFactory) WithConfig(provide func(cfg config.CfgManager)) AppFactory {
	provide(af.cfg)
	return af
}

func (af *ApplicationFactory) Get(typ reflect.Type) any {
	return af.apps[typ]
}

func GetApp[T any]() T {
	return Factory.Get(reflect.TypeFor[T]()).(T)
}
