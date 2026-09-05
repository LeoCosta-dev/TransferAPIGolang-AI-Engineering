package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
)

const stubAccountID = "550e8400-e29b-41d4-a716-446655440000"

type serviceFake struct {
	registerErr  error
	changeErr    error
	creditResult domain.Transaction
	creditErr    error
	debitResult  domain.Transaction
	debitErr     error
	balance      int64
	balanceErr   error

	lastKey       string
	lastAccountID string
}

func (f *serviceFake) Register(_ context.Context, _ string, _ domain.Status) error {
	return f.registerErr
}

func (f *serviceFake) ChangeStatus(_ context.Context, _ string, _ domain.Status, _ domain.Status) error {
	return f.changeErr
}

func (f *serviceFake) Credit(_ context.Context, accountID string, _ int64, key string) (domain.Transaction, error) {
	f.lastAccountID, f.lastKey = accountID, key
	if f.creditErr != nil {
		return domain.Transaction{}, f.creditErr
	}
	return f.creditResult, nil
}

func (f *serviceFake) Debit(_ context.Context, accountID string, _ int64, key string) (domain.Transaction, error) {
	f.lastAccountID, f.lastKey = accountID, key
	if f.debitErr != nil {
		return domain.Transaction{}, f.debitErr
	}
	return f.debitResult, nil
}

func (f *serviceFake) Balance(_ context.Context, accountID string) (int64, error) {
	f.lastAccountID = accountID
	if f.balanceErr != nil {
		return 0, f.balanceErr
	}
	return f.balance, nil
}

func newServer(service service) *echo.Echo {
	e := echo.New()
	RegisterRoutes(e, NewHandler(service))
	return e
}

func doRequest(e *echo.Echo, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func jsonContentType() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Idempotency-Key": "key-1"}
}

func responseErrorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON de erro: %q", rec.Body.String())
	}
	return payload.Error
}

func TestCreditEndpointReturnsOperation(t *testing.T) {
	fake := &serviceFake{creditResult: domain.Transaction{
		ID: stubAccountID + ":key-1", AccountID: stubAccountID, Type: domain.TypeCredit, Amount: 250, Balance: 250,
	}}
	e := newServer(fake)

	rec := doRequest(e, http.MethodPost, "/api/v1/transactions/"+stubAccountID+"/credits", `{"amount":250}`, jsonContentType())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["id"] != stubAccountID+":key-1" || payload["account_id"] != stubAccountID || payload["type"] != "CREDIT" {
		t.Fatalf("payload = %v", payload)
	}
	if payload["amount"] != float64(250) || payload["balance"] != float64(250) {
		t.Fatalf("payload = %v", payload)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if fake.lastKey != "key-1" {
		t.Fatalf("Idempotency-Key repassada = %q", fake.lastKey)
	}
	if fake.lastAccountID != stubAccountID {
		t.Fatalf("account id repassado = %q", fake.lastAccountID)
	}
}

func TestCreditEndpointErrorResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		headers    map[string]string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"amount ausente", `{}`, jsonContentType(), nil, http.StatusBadRequest, "campo amount obrigatório"},
		{"amount nulo", `{"amount":null}`, jsonContentType(), nil, http.StatusBadRequest, "campo amount obrigatório"},
		{"json inválido", `{"amount":`, jsonContentType(), nil, http.StatusBadRequest, "JSON inválido"},
		{"campo desconhecido", `{"amount":100,"extra":1}`, jsonContentType(), nil, http.StatusBadRequest, "JSON inválido"},
		{"content-type inválido", `{"amount":100}`, map[string]string{"Content-Type": "text/plain"}, nil, http.StatusBadRequest, "Content-Type deve ser application/json"},
		{"idempotency key inválida", `{"amount":100}`, jsonContentType(), application.ErrInvalidIdempotencyKey, http.StatusBadRequest, "Idempotency-Key inválido"},
		{"valor inválido", `{"amount":0}`, jsonContentType(), domain.ErrInvalidAmount, http.StatusBadRequest, "valor monetário inválido"},
		{"conta inexistente", `{"amount":100}`, jsonContentType(), application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
		{"saldo insuficiente", `{"amount":100}`, jsonContentType(), domain.ErrInsufficientBalance, http.StatusConflict, "saldo insuficiente"},
		{"conta bloqueada", `{"amount":100}`, jsonContentType(), domain.ErrAccountBlocked, http.StatusConflict, "conta bloqueada"},
		{"conta fechada", `{"amount":100}`, jsonContentType(), domain.ErrAccountClosed, http.StatusConflict, "conta fechada"},
		{"conflito de idempotência", `{"amount":100}`, jsonContentType(), application.ErrIdempotencyConflict, http.StatusConflict, "conflito de idempotência"},
		{"erro interno", `{"amount":100}`, jsonContentType(), errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&serviceFake{creditErr: tc.serviceErr})
			rec := doRequest(e, http.MethodPost, "/api/v1/transactions/"+stubAccountID+"/credits", tc.body, tc.headers)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := responseErrorMessage(t, rec); got != tc.wantError {
				t.Fatalf("erro = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestDebitEndpointReturnsOperation(t *testing.T) {
	fake := &serviceFake{debitResult: domain.Transaction{
		ID: stubAccountID + ":key-1", AccountID: stubAccountID, Type: domain.TypeDebit, Amount: 50, Balance: 200,
	}}
	e := newServer(fake)

	rec := doRequest(e, http.MethodPost, "/api/v1/transactions/"+stubAccountID+"/debits", `{"amount":50}`, jsonContentType())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["type"] != "DEBIT" || payload["amount"] != float64(50) || payload["balance"] != float64(200) {
		t.Fatalf("payload = %v", payload)
	}
	if fake.lastKey != "key-1" {
		t.Fatalf("Idempotency-Key repassada = %q", fake.lastKey)
	}
}

func TestDebitEndpointErrorResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"amount ausente", `{}`, nil, http.StatusBadRequest, "campo amount obrigatório"},
		{"saldo insuficiente", `{"amount":100}`, domain.ErrInsufficientBalance, http.StatusConflict, "saldo insuficiente"},
		{"erro interno", `{"amount":100}`, errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&serviceFake{debitErr: tc.serviceErr})
			rec := doRequest(e, http.MethodPost, "/api/v1/transactions/"+stubAccountID+"/debits", tc.body, jsonContentType())
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := responseErrorMessage(t, rec); got != tc.wantError {
				t.Fatalf("erro = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestBalanceEndpointReturnsBalance(t *testing.T) {
	e := newServer(&serviceFake{balance: 77})

	rec := doRequest(e, http.MethodGet, "/api/v1/transactions/"+stubAccountID+"/balance", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["account_id"] != stubAccountID || payload["balance"] != float64(77) {
		t.Fatalf("payload = %v", payload)
	}
	if _, exists := payload["amount"]; exists {
		t.Fatal("resposta de saldo não deve incluir amount")
	}
	if _, exists := payload["type"]; exists {
		t.Fatal("resposta de saldo não deve incluir type")
	}
}

func TestBalanceEndpointErrorResponses(t *testing.T) {
	cases := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"conta inexistente", application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
		{"erro interno", errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&serviceFake{balanceErr: tc.serviceErr})
			rec := doRequest(e, http.MethodGet, "/api/v1/transactions/"+stubAccountID+"/balance", "", nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := responseErrorMessage(t, rec); got != tc.wantError {
				t.Fatalf("erro = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestRegisterEndpointResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"registro válido", `{"status":"ACTIVE"}`, nil, http.StatusCreated, ""},
		{"status inválido", `{"status":"PAUSED"}`, nil, http.StatusBadRequest, "status inválido"},
		{"status ausente", `{}`, nil, http.StatusBadRequest, "status inválido"},
		{"json inválido", `{"status":`, nil, http.StatusBadRequest, "JSON inválido"},
		{"erro do serviço", `{"status":"ACTIVE"}`, application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&serviceFake{registerErr: tc.serviceErr})
			rec := doRequest(e, http.MethodPost, "/internal/v1/accounts/"+stubAccountID+"/register", tc.body, jsonContentType())
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := responseErrorMessage(t, rec); got != tc.wantError {
					t.Fatalf("erro = %q, want %q", got, tc.wantError)
				}
				return
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("corpo = %q, want vazio", rec.Body.String())
			}
		})
	}
}

func TestChangeStatusEndpointResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"mudança válida", `{"from":"ACTIVE","status":"BLOCKED"}`, nil, http.StatusOK, ""},
		{"from ausente", `{"status":"BLOCKED"}`, nil, http.StatusBadRequest, "status inválido"},
		{"status inválido", `{"from":"ACTIVE","status":"PAUSED"}`, nil, http.StatusBadRequest, "status inválido"},
		{"json inválido", `{"from":`, nil, http.StatusBadRequest, "JSON inválido"},
		{"conta inexistente", `{"from":"ACTIVE","status":"BLOCKED"}`, application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
		{"erro interno", `{"from":"ACTIVE","status":"BLOCKED"}`, errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&serviceFake{changeErr: tc.serviceErr})
			rec := doRequest(e, http.MethodPost, "/internal/v1/accounts/"+stubAccountID+"/status", tc.body, jsonContentType())
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := responseErrorMessage(t, rec); got != tc.wantError {
					t.Fatalf("erro = %q, want %q", got, tc.wantError)
				}
				return
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("corpo = %q, want vazio", rec.Body.String())
			}
		})
	}
}

func TestHealthEndpointReturnsOK(t *testing.T) {
	e := newServer(&serviceFake{})

	rec := doRequest(e, http.MethodGet, "/health", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["status"] != "ok" {
		t.Fatalf("payload = %v", payload)
	}
}
