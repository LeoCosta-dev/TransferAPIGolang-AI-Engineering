package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/persistence/sqlite"
	httpapi "github.com/lcosta/TransferAPIGolang/services/account/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	databasePath := os.Getenv("ACCOUNT_DB_PATH")
	if databasePath == "" {
		databasePath = "account.db"
	}
	address := os.Getenv("ACCOUNT_HTTP_ADDR")
	if address == "" {
		address = ":8088"
	}

	repository, err := sqlite.Open(ctx, databasePath)
	if err != nil {
		return err
	}
	defer repository.Close()

	service := application.NewService(repository)
	e := echo.New()
	httpapi.RegisterRoutes(e, httpapi.NewHandler(service))
	server := &http.Server{Addr: address, Handler: e}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	shutdownContext, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-shutdownContext.Done():
		shutdownDeadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownDeadline); err != nil {
			return err
		}
		return nil
	}
}
