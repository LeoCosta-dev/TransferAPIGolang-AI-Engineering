package application

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

func TestNewHTTPFinancialGatewayRequiresBaseURL(t *testing.T) {
	if _, err := NewHTTPFinancialGateway("   ", nil); err == nil || !strings.Contains(err.Error(), "TRANSACTIONS_SERVICE_URL") {
		t.Fatalf("erro = %v", err)
	}
}

func TestGatewayRegisterPostsFinancialAccount(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := gateway.Register(context.Background(), stubAccountID, domain.StatusActive); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("método = %q, want POST", gotMethod)
	}
	if gotPath != "/internal/v1/accounts/"+stubAccountID+"/register" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if gotBody != `{"status":"ACTIVE"}` {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestGatewayRegisterMapsConflictToAccountHasBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gateway.Register(context.Background(), stubAccountID, domain.StatusActive)
	if !errors.Is(err, ErrAccountHasBalance) {
		t.Fatalf("erro = %v, want ErrAccountHasBalance", err)
	}
}

func TestGatewayRegisterRejectsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gateway.Register(context.Background(), stubAccountID, domain.StatusActive)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("erro = %v, want contendo 'status 500'", err)
	}
}

func TestGatewayChangeStatusPostsTransition(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := gateway.ChangeStatus(context.Background(), stubAccountID, domain.StatusActive, domain.StatusClosed); err != nil {
		t.Fatal(err)
	}

	if gotPath != "/internal/v1/accounts/"+stubAccountID+"/status" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody != `{"from":"ACTIVE","status":"CLOSED"}` {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestGatewayChangeStatusMapsConflictToAccountHasBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gateway.ChangeStatus(context.Background(), stubAccountID, domain.StatusActive, domain.StatusClosed)
	if !errors.Is(err, ErrAccountHasBalance) {
		t.Fatalf("erro = %v, want ErrAccountHasBalance", err)
	}
}

func TestGatewayChangeStatusRejectsUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gateway.ChangeStatus(context.Background(), stubAccountID, domain.StatusActive, domain.StatusClosed)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("erro = %v, want contendo 'status 404'", err)
	}
}

func TestGatewayPropagatesConnectionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	gateway, err := NewHTTPFinancialGateway(url, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gateway.Register(context.Background(), stubAccountID, domain.StatusActive)
	if err == nil || !strings.Contains(err.Error(), "sincronizar estado financeiro") {
		t.Fatalf("erro = %v, want contendo 'sincronizar estado financeiro'", err)
	}
}

// O cliente injetado com timeout curto contra um servidor lento comprova que
// o gateway respeita o timeout configurado (comportamento introduzido para
// impedir requisições presas quando o Transactions Service não responde).
func TestGatewayAppliesClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	gateway, err := NewHTTPFinancialGateway(server.URL, &http.Client{Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	if err := gateway.Register(context.Background(), stubAccountID, domain.StatusActive); err == nil {
		t.Fatal("esperado erro de timeout do cliente")
	}
}
