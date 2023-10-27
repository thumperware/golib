package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thumperq/golib/config"
)

type ApiServer struct {
	HttpPort      uint16
	Engine        *gin.Engine
	ConfigManager config.CfgManager
	interrupt     chan os.Signal
	httpServer    *http.Server
}

func (srv *ApiServer) Initialize() error {
	srv.Engine = gin.New()
	srv.interrupt = make(chan os.Signal, 1)
	signal.Notify(srv.interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return nil
}

func (srv *ApiServer) Start() error {
	srv.httpServer = &http.Server{
		Handler:           srv.Engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.HttpPort))
	if err != nil {
		return err
	}
	go func() {
		if err := srv.httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return nil
}

func (srv *ApiServer) Stop() int {
	<-srv.interrupt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.httpServer.Shutdown(ctx); err != nil {
		return 1
	}
	return 1
}

func ListenAndServe(callback func(*ApiServer)) <-chan int {
	var httpPort uint16 = 8080
	configManager := config.NewConfigManager()
	exitCode := make(chan int, 1)
	apiServer := ApiServer{
		HttpPort:      httpPort,
		ConfigManager: configManager,
	}
	err := apiServer.Initialize()
	if err != nil {
		exitCode <- 1
		close(exitCode)
		return exitCode
	}
	callback(&apiServer)
	err = apiServer.Start()
	if err != nil {
		exitCode <- 1
		close(exitCode)
		return exitCode
	}
	go func() {
		exitCode <- apiServer.Stop()
		close(exitCode)
	}()
	return exitCode
}
