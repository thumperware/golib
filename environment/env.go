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

	dbFactory, err := database.NewDBFactory(cfg)
	if err != nil {
		return nil, err
	}

	domain := os.Getenv("DOMAIN")
	service := os.Getenv("SERVICE")

	broker, err := messaging.NewBroker(cfg, domain, service)

	if err != nil {
		return nil, err
	}

	appFactory := application.NewApplicationFactory()

	return &Env{
		Broker:     broker,
		AppFactory: appFactory,
		DbFactory:  dbFactory,
		Cfg:        cfg,
	}, nil
}

func (env *Env) Bootstrap(b func(env *Env, apiSrv *httpserver.ApiServer) error) error {
	err := env.Broker.Connect()
	if err != nil {
		return err
	}
	exitCode := <-httpserver.ListenAndServe(func(as *httpserver.ApiServer) error {
		return b(env, as)
	})
	err = env.Broker.Disconnect()
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
