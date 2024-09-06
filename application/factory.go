package application

import (
	"reflect"
)

type AppFactory interface {
	Register(newApp func(AppFactory) any) AppFactory
	Get(typ reflect.Type) any
}

type appFactory struct {
	apps map[reflect.Type]any
}

func NewAppFactory() AppFactory {
	return &appFactory{
		apps: make(map[reflect.Type]any),
	}
}

func (af *appFactory) Register(newApp func(AppFactory) any) AppFactory {
	app := newApp(af)
	af.apps[reflect.TypeOf(app)] = app
	return af
}

func (af *appFactory) Get(typ reflect.Type) any {
	return af.apps[typ]
}
