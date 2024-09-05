package environment

import (
	"context"
	"os"

	"github.com/thumperq/golib/application"
	"github.com/thumperq/golib/config"
	"github.com/thumperq/golib/database"
	"github.com/thumperq/golib/logging"
	"github.com/thumperq/golib/messaging"
	httpserver "github.com/thumperq/golib/servers/http"
)

type Env struct {
	broker     *messaging.Broker
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

	dbf, err := database.NewDBFactory(cfg)
	if err != nil {
		return nil, err
	}

	domain := os.Getenv("DOMAIN")
	service := os.Getenv("SERVICE")

	broker, err := messaging.NewBroker(cfg, domain, service)

	if err != nil {
		return nil, err
	}

	return &Env{
		broker:     broker,
		AppFactory: application.NewApplicationFactory(dbf, broker, cfg),
		DbFactory:  dbf,
		Cfg:        cfg,
	}, nil
}

func (env *Env) Bootstrap(b func(env *Env, apiSrv *httpserver.ApiServer) error) error {
	exitCode := <-httpserver.ListenAndServe(func(as *httpserver.ApiServer) error {
		return b(env, as)
	})
	err := env.broker.Disconnect()
	if err != nil {
		logging.TraceLogger(context.Background()).
			Err(err).
			Msg("error disconnecting from broker")
	}
	os.Exit(exitCode)
	return err
}
