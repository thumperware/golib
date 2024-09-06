package environment

import (
	"context"
	"os"
	"reflect"

	"github.com/thumperq/golib/application"
	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/database"
	"github.com/thumperq/golib/logging"
	"github.com/thumperq/golib/messaging"
	httpserver "github.com/thumperq/golib/servers/http"
)

var appFactory application.AppFactory
var dbFactory database.DbFactory

type Env struct {
	providers  []func(*Env) error
	Broker     *messaging.Broker
	AppFactory application.AppFactory
	DbFactory  database.DbFactory
	Cfg        config.CfgManager
}

func NewEnv() (*Env, error) {
	logging.SetupLogging()
	cfg, err := config.NewConfigManager()
	if err != nil {
		return nil, err
	}
	return &Env{
		Cfg: cfg,
	}, nil
}

func (env *Env) WithBroker() *Env {
	env.providers = append(env.providers, func(env *Env) error {
		domain := os.Getenv("DOMAIN")
		service := os.Getenv("SERVICE")
		broker, err := messaging.NewBroker(env.Cfg, domain, service)
		if err != nil {
			return err
		}
		env.Broker = broker
		err = env.Broker.Connect()
		if err != nil {
			return err
		}
		return nil
	})
	return env
}

func (env *Env) WithDbFactory() *Env {
	env.providers = append(env.providers, func(env *Env) error {
		dbf, err := database.NewDBFactory(env.Cfg)
		if err != nil {
			return err
		}
		dbFactory = dbf
		env.DbFactory = dbFactory
		return nil
	})
	return env
}

func (env *Env) WithAppFactory() *Env {
	env.providers = append(env.providers, func(env *Env) error {
		appFactory = application.NewApplicationFactory()
		env.AppFactory = appFactory
		return nil
	})
	return env
}

func (env *Env) Bootstrap(b func(env *Env, apiSrv *httpserver.ApiServer) error) error {
	for _, provider := range env.providers {
		err := provider(env)
		if err != nil {
			return err
		}
	}
	exitCode := <-httpserver.ListenAndServe(func(as *httpserver.ApiServer) error {
		return b(env, as)
	})
	err := env.Broker.Disconnect()
	if err != nil {
		logging.TraceLogger(context.Background()).
			Err(err).
			Msg("error disconnecting from broker")
	}
	os.Exit(exitCode)
	return err
}

func GetApp[T any]() T {
	return appFactory.Get(reflect.TypeFor[T]()).(T)
}

func GetRepo[T any]() T {
	return dbFactory.Get(reflect.TypeFor[T]()).(T)
}
