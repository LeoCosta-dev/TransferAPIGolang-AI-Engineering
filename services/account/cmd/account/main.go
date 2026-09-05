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

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/persistence/mongodb"
	httpapi "github.com/lcosta/TransferAPIGolang/services/account/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("carregar arquivo de configuração local")
	}
	uri, database := os.Getenv("MONGODB_URI"), os.Getenv("MONGODB_DATABASE")
	if uri == "" || database == "" {
		return errors.New("MONGODB_URI e MONGODB_DATABASE são obrigatórios")
	}
	ctx := context.Background()
	repository, err := mongodb.Open(ctx, uri, database)
	if err != nil {
		return err
	}
	defer repository.Close(context.Background())
	financial, err := application.NewHTTPFinancialGateway(os.Getenv("TRANSACTIONS_SERVICE_URL"), nil)
	if err != nil {
		return err
	}
	service := application.NewService(repository, financial)
	e := echo.New()
	httpapi.RegisterRoutes(e, httpapi.NewHandler(service))
	address := os.Getenv("ACCOUNT_HTTP_ADDR")
	if address == "" {
		address = ":8088"
	}
	server := &http.Server{
		Addr:              address,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Account Service rodando em %s", address)
		serverErrors <- server.ListenAndServe()
	}()
	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownContext.Done():
		deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(deadline)
	}
}
