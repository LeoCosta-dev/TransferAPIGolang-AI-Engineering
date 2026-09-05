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
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/persistence/mongodb"
	httpapi "github.com/lcosta/TransferAPIGolang/services/transactions/internal/transport/http"
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
	repository, err := mongodb.Open(context.Background(), uri, database)
	if err != nil {
		return err
	}
	defer repository.Close(context.Background())
	e := echo.New()
	httpapi.RegisterRoutes(e, httpapi.NewHandler(application.NewService(repository)))
	address := os.Getenv("TRANSACTIONS_HTTP_ADDR")
	if address == "" {
		address = ":8089"
	}
	server := &http.Server{Addr: address, Handler: e}
	errorsChannel := make(chan error, 1)
	go func() {
		log.Printf("Transactions Service rodando em %s", address)
		errorsChannel <- server.ListenAndServe()
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-errorsChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(deadline)
	}
}
