package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

const testAccountID = "550e8400-e29b-41d4-a716-446655440000"

type accountServiceFake struct {
	account      domain.Account
	accountError error
	operationErr error
	createCalls  int
	updateCalls  int
	creditCalls  int
	debitCalls   int
	lastAmount   int64
	lastStatus   domain.Status
	lastName     string
}

func (fake *accountServiceFake) CreateAccount(_ context.Context, name, document string) (domain.Account, error) {
	fake.createCalls++
	if fake.operationErr != nil {
		return domain.Account{}, fake.operationErr
	}
	fake.account.Name = name
	fake.account.Document = document
	return fake.account, nil
}

func (fake *accountServiceFake) GetAccount(_ context.Context, _ string) (domain.Account, error) {
	if fake.accountError != nil {
		return domain.Account{}, fake.accountError
	}
	return fake.account, nil
}

func (fake *accountServiceFake) UpdateName(_ context.Context, _, name string) (domain.Account, error) {
	fake.updateCalls++
	fake.lastName = name
	if fake.operationErr != nil {
		return domain.Account{}, fake.operationErr
	}
	fake.account.Name = name
	return fake.account, nil
}

func (fake *accountServiceFake) ChangeStatus(_ context.Context, _ string, status domain.Status) (domain.Account, error) {
	fake.lastStatus = status
	if fake.operationErr != nil {
		return domain.Account{}, fake.operationErr
	}
	fake.account.Status = status
	return fake.account, nil
}

func (fake *accountServiceFake) GetBalance(_ context.Context, _ string) (int64, error) {
	if fake.accountError != nil {
		return 0, fake.accountError
	}
	return fake.account.Balance, nil
}

func (fake *accountServiceFake) Credit(_ context.Context, _ string, amount int64) (domain.Account, error) {
	fake.creditCalls++
	fake.lastAmount = amount
	if fake.operationErr != nil {
		return domain.Account{}, fake.operationErr
	}
	fake.account.Balance += amount
	return fake.account, nil
}

func (fake *accountServiceFake) Debit(_ context.Context, _ string, amount int64) (domain.Account, error) {
	fake.debitCalls++
	fake.lastAmount = amount
	if fake.operationErr != nil {
		return domain.Account{}, fake.operationErr
	}
	fake.account.Balance -= amount
	return fake.account, nil
}

func newHTTPTestServer(service *accountServiceFake) *echo.Echo {
	e := echo.New()
	RegisterRoutes(e, NewHandler(service))
	return e
}

func newHTTPTestAccount() domain.Account {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	return domain.Account{
		ID:        testAccountID,
		Name:      "Nome",
		Document:  "123",
		Balance:   100,
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func request(t *testing.T, server *echo.Echo, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	return recorder
}

func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

func TestAccountRoutes(t *testing.T) {
	service := &accountServiceFake{account: newHTTPTestAccount()}
	server := newHTTPTestServer(service)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCall   func()
	}{
		{"health", http.MethodGet, "/health", "", http.StatusOK, func() {}},
		{"create", http.MethodPost, "/api/v1/accounts", `{"name":"Novo","document":"456"}`, http.StatusCreated, func() {
			if service.createCalls != 1 {
				t.Errorf("createCalls = %d, want 1", service.createCalls)
			}
		}},
		{"get", http.MethodGet, "/api/v1/accounts/" + testAccountID, "", http.StatusOK, func() {}},
		{"update", http.MethodPatch, "/api/v1/accounts/" + testAccountID, `{"name":"Atualizado"}`, http.StatusOK, func() {
			if service.lastName != "Atualizado" {
				t.Errorf("lastName = %q", service.lastName)
			}
		}},
		{"status", http.MethodPatch, "/api/v1/accounts/" + testAccountID + "/status", `{"status":"BLOCKED"}`, http.StatusOK, func() {
			if service.lastStatus != domain.StatusBlocked {
				t.Errorf("lastStatus = %q", service.lastStatus)
			}
		}},
		{"balance", http.MethodGet, "/api/v1/accounts/" + testAccountID + "/balance", "", http.StatusOK, func() {}},
		{"credit", http.MethodPost, "/api/v1/accounts/" + testAccountID + "/credits", `{"amount":25}`, http.StatusOK, func() {
			if service.creditCalls != 1 || service.lastAmount != 25 {
				t.Errorf("credit calls=%d amount=%d", service.creditCalls, service.lastAmount)
			}
		}},
		{"debit", http.MethodPost, "/api/v1/accounts/" + testAccountID + "/debits", `{"amount":10}`, http.StatusOK, func() {
			if service.debitCalls != 1 || service.lastAmount != 10 {
				t.Errorf("debit calls=%d amount=%d", service.debitCalls, service.lastAmount)
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers := map[string]string{}
			if test.body != "" {
				headers = jsonHeaders()
			}
			if test.name == "credit" || test.name == "debit" {
				headers["Idempotency-Key"] = "operation-1"
			}
			response := request(t, server, test.method, test.path, test.body, headers)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			test.wantCall()
		})
	}
}

func TestJSONValidation(t *testing.T) {
	server := newHTTPTestServer(&accountServiceFake{account: newHTTPTestAccount()})
	for _, test := range []struct {
		name    string
		body    string
		headers map[string]string
	}{
		{"missing content type", `{"name":"Nome","document":"123"}`, map[string]string{}},
		{"unknown field", `{"name":"Nome","document":"123","id":"forbidden"}`, jsonHeaders()},
		{"malformed json", `{"name":`, jsonHeaders()},
		{"missing name", `{"document":"123"}`, jsonHeaders()},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, server, http.MethodPost, "/api/v1/accounts", test.body, test.headers)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestIdempotencyKeyIsValidatedButNotStored(t *testing.T) {
	service := &accountServiceFake{account: newHTTPTestAccount()}
	server := newHTTPTestServer(service)

	for _, key := range []string{"", "   ", strings.Repeat("a", 256)} {
		response := request(t, server, http.MethodPost, "/api/v1/accounts/"+testAccountID+"/credits", `{"amount":1}`, map[string]string{
			"Content-Type":    "application/json",
			"Idempotency-Key": key,
		})
		if response.Code != http.StatusBadRequest {
			t.Errorf("key length %d: status = %d, want %d", len(key), response.Code, http.StatusBadRequest)
		}
	}

	first := request(t, server, http.MethodPost, "/api/v1/accounts/"+testAccountID+"/credits", `{"amount":1}`, map[string]string{
		"Content-Type":    "application/json",
		"Idempotency-Key": "same-key",
	})
	second := request(t, server, http.MethodPost, "/api/v1/accounts/"+testAccountID+"/credits", `{"amount":1}`, map[string]string{
		"Content-Type":    "application/json",
		"Idempotency-Key": "same-key",
	})
	if first.Code != http.StatusOK || second.Code != http.StatusOK || service.creditCalls != 2 {
		t.Fatalf("idempotência foi implementada nesta etapa: first=%d second=%d calls=%d", first.Code, second.Code, service.creditCalls)
	}
}

func TestApplicationErrorsAreMapped(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", application.ErrAccountNotFound, http.StatusNotFound},
		{"duplicate", application.ErrDuplicateDocument, http.StatusConflict},
		{"invalid amount", domain.ErrInvalidAmount, http.StatusBadRequest},
		{"business conflict", domain.ErrInsufficientBalance, http.StatusConflict},
		{"unexpected", errors.New("database details"), http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &accountServiceFake{account: newHTTPTestAccount(), accountError: test.err, operationErr: test.err}
			server := newHTTPTestServer(service)
			response := request(t, server, http.MethodGet, "/api/v1/accounts/"+testAccountID, "", nil)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"error"`) {
				t.Fatalf("resposta sem campo error: %s", response.Body.String())
			}
		})
	}
}
