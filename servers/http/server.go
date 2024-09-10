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

	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

type ApiServer struct {
	HttpPort   uint16
	Engine     *http.ServeMux
	interrupt  chan os.Signal
	httpServer *http.Server
}

func (srv *ApiServer) initialize() error {
	srv.Engine = http.NewServeMux()
	domain := os.Getenv("DOMAIN")
	service := os.Getenv("SERVICE")
	srv.Engine.HandleFunc(fmt.Sprintf("GET /%s/%s/health-check", domain, service), func(w http.ResponseWriter, r *http.Request) {
		Json(http.StatusOK, w, H{
			"status": "ok",
		})
	})

	// Serve the OpenAPI spec at /openapi.yaml
	srv.Engine.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../api/openapi.yaml")
	})

	// Serve Swagger UI at /swagger/
	srv.Engine.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/openapi.yaml"), // The URL where the OpenAPI YAML file is served
	))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// Enable Debugging for testing, consider disabling in production
		// Debug: true,
	})

	c.Handler(srv.Engine)

	srv.interrupt = make(chan os.Signal, 1)
	signal.Notify(srv.interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return nil
}

func (srv *ApiServer) start() error {
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

func (srv *ApiServer) stop() int {
	<-srv.interrupt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.httpServer.Shutdown(ctx); err != nil {
		return 1
	}
	return 1
}

func ListenAndServe(callback func(*ApiServer) error) <-chan int {
	var httpPort uint16 = 8080
	exitCode := make(chan int, 1)
	apiServer := ApiServer{
		HttpPort: httpPort,
	}
	err := apiServer.initialize()
	if err != nil {
		exitCode <- 1
		close(exitCode)
		return exitCode
	}
	err = callback(&apiServer)
	if err != nil {
		exitCode <- 1
		close(exitCode)
		return exitCode
	}
	err = apiServer.start()
	if err != nil {
		exitCode <- 1
		close(exitCode)
		return exitCode
	}
	go func() {
		exitCode <- apiServer.stop()
		close(exitCode)
	}()
	return exitCode
}
