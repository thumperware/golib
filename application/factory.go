package application

import (
	"github.com/thumperq/golib/database"
	"github.com/thumperq/golib/messaging"
)

type AppService interface {
	Name() string
}

type AppFactory interface {
	DbFactory() database.DbFactory
	Broker() *messaging.Broker
}

type ApplicationFactory struct {
	dbFactory database.DbFactory
	broker    *messaging.Broker
	apps      map[string]AppService
}

func NewApplicationFactory(dbFactory database.DbFactory, broker *messaging.Broker) *ApplicationFactory {
	return &ApplicationFactory{
		dbFactory: dbFactory,
		broker:    broker,
	}
}

func (af *ApplicationFactory) Register(newApp func(...any) AppService) {
	app := newApp(af.dbFactory, af.broker)
	af.apps[app.Name()] = app
}

func (af *ApplicationFactory) DbFactory() database.DbFactory {
	return af.dbFactory
}

func (af *ApplicationFactory) Broker() *messaging.Broker {
	return af.broker
}
