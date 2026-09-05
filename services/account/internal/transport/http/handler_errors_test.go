package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

const stubAccountID = "550e8400-e29b-41d4-a716-446655440000"

type stubService struct {
	account   domain.Account
	createErr error
	getErr    error
	updateErr error
	changeErr error
}

func (service *stubService) CreateAccount(_ context.Context, name, document string) (domain.Account, error) {
	if service.createErr != nil {
		return domain.Account{}, service.createErr
	}
	service.account.Name, service.account.Document = name, document
	return service.account, nil
}

func (service *stubService) GetAccount(_ context.Context, _ string) (domain.Account, error) {
	if service.getErr != nil {
		return domain.Account{}, service.getErr
	}
	return service.account, nil
}

func (service *stubService) UpdateName(_ context.Context, _ string, name string) (domain.Account, error) {
	if service.updateErr != nil {
		return domain.Account{}, service.updateErr
	}
	service.account.Name = name
	return service.account, nil
}

func (service *stubService) ChangeStatus(_ context.Context, _ string, status domain.Status) (domain.Account, error) {
	if service.changeErr != nil {
		return domain.Account{}, service.changeErr
	}
	service.account.Status = status
	return service.account, nil
}

func executeRequest(e *echo.Echo, method, target, body, contentType string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
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

func TestCreateAccountReturnsCreatedAccount(t *testing.T) {
	e := newServer(&stubService{account: domain.Account{Status: domain.StatusActive}})

	rec := executeRequest(e, http.MethodPost, "/api/v1/accounts", `{"name":"Nome","document":"123"}`, "application/json")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["name"] != "Nome" || payload["document"] != "123" || payload["status"] != "ACTIVE" {
		t.Fatalf("payload = %v", payload)
	}
	if _, exists := payload["balance"]; exists {
		t.Fatal("resposta de conta não deve expor saldo")
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestCreateAccountErrorResponses(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		contentType string
		serviceErr  error
		wantStatus  int
		wantError   string
	}{
		{"content-type ausente", `{"name":"Nome","document":"123"}`, "", nil, http.StatusBadRequest, "Content-Type deve ser application/json"},
		{"content-type inválido", `{"name":"Nome","document":"123"}`, "text/plain", nil, http.StatusBadRequest, "Content-Type deve ser application/json"},
		{"json inválido", `{`, "application/json", nil, http.StatusBadRequest, "JSON inválido"},
		{"campo desconhecido", `{"name":"Nome","document":"123","extra":true}`, "application/json", nil, http.StatusBadRequest, "JSON inválido"},
		{"múltiplos objetos", `{"name":"Nome","document":"123"}{"name":"Outro","document":"456"}`, "application/json", nil, http.StatusBadRequest, "JSON deve conter apenas um objeto"},
		{"campos obrigatórios ausentes", `{}`, "application/json", nil, http.StatusBadRequest, "campos obrigatórios ausentes"},
		{"documento duplicado", `{"name":"Nome","document":"123"}`, "application/json", application.ErrDuplicateDocument, http.StatusConflict, "documento já associado a uma conta"},
		{"nome inválido", `{"name":"Nome","document":"123"}`, "application/json", domain.ErrInvalidName, http.StatusBadRequest, "nome inválido"},
		{"erro interno", `{"name":"Nome","document":"123"}`, "application/json", errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&stubService{createErr: tc.serviceErr})
			rec := executeRequest(e, http.MethodPost, "/api/v1/accounts", tc.body, tc.contentType)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := responseErrorMessage(t, rec); got != tc.wantError {
				t.Fatalf("erro = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestGetAccountErrorResponses(t *testing.T) {
	cases := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"conta inexistente", application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
		{"identificador inválido", domain.ErrInvalidAccountID, http.StatusBadRequest, "identificador de conta inválido"},
		{"erro interno", errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&stubService{getErr: tc.serviceErr})
			rec := executeRequest(e, http.MethodGet, "/api/v1/accounts/"+stubAccountID, "", "")
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := responseErrorMessage(t, rec); got != tc.wantError {
				t.Fatalf("erro = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestGetAccountReturnsAccountWithoutBalance(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	service := &stubService{account: domain.Account{
		ID: stubAccountID, Name: "Nome", Document: "123", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}}
	e := newServer(service)

	rec := executeRequest(e, http.MethodGet, "/api/v1/accounts/"+stubAccountID, "", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("corpo não é JSON: %q", rec.Body.String())
	}
	if payload["id"] != stubAccountID || payload["name"] != "Nome" || payload["document"] != "123" || payload["status"] != "ACTIVE" {
		t.Fatalf("payload = %v", payload)
	}
	if _, exists := payload["balance"]; exists {
		t.Fatal("resposta de conta não deve expor saldo")
	}
}

func TestUpdateAccountResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"atualização válida", `{"name":"Novo"}`, nil, http.StatusOK, ""},
		{"name ausente", `{"name":null}`, nil, http.StatusBadRequest, "campo name obrigatório"},
		{"json inválido", `{"name"`, nil, http.StatusBadRequest, "JSON inválido"},
		{"nome inválido", `{"name":"   "}`, domain.ErrInvalidName, http.StatusBadRequest, "nome inválido"},
		{"conta inexistente", `{"name":"Novo"}`, application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
		{"erro interno", `{"name":"Novo"}`, errors.New("falha inesperada"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&stubService{updateErr: tc.serviceErr})
			rec := executeRequest(e, http.MethodPatch, "/api/v1/accounts/"+stubAccountID, tc.body, "application/json")
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := responseErrorMessage(t, rec); got != tc.wantError {
					t.Fatalf("erro = %q, want %q", got, tc.wantError)
				}
				return
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("corpo não é JSON: %q", rec.Body.String())
			}
			if payload["name"] != "Novo" {
				t.Fatalf("payload = %v", payload)
			}
		})
	}
}

func TestChangeAccountStatusResponses(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{"mudança válida", `{"status":"BLOCKED"}`, nil, http.StatusOK, ""},
		{"status ausente", `{}`, nil, http.StatusBadRequest, "status inválido"},
		{"status inválido", `{"status":"PAUSED"}`, nil, http.StatusBadRequest, "status inválido"},
		{"json inválido", `{"status":`, nil, http.StatusBadRequest, "JSON inválido"},
		{"transição inválida", `{"status":"BLOCKED"}`, domain.ErrInvalidStatusChange, http.StatusConflict, "transição de status não permitida"},
		{"conta com saldo", `{"status":"CLOSED"}`, application.ErrAccountHasBalance, http.StatusConflict, "conta possui saldo"},
		{"conta inexistente", `{"status":"BLOCKED"}`, application.ErrAccountNotFound, http.StatusNotFound, "conta não encontrada"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newServer(&stubService{changeErr: tc.serviceErr})
			rec := executeRequest(e, http.MethodPatch, "/api/v1/accounts/"+stubAccountID+"/status", tc.body, "application/json")
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := responseErrorMessage(t, rec); got != tc.wantError {
					t.Fatalf("erro = %q, want %q", got, tc.wantError)
				}
				return
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("corpo não é JSON: %q", rec.Body.String())
			}
			if payload["status"] != "BLOCKED" {
				t.Fatalf("payload = %v", payload)
			}
		})
	}
}

func TestHealthReturnsOK(t *testing.T) {
	e := newServer(&stubService{})

	rec := executeRequest(e, http.MethodGet, "/health", "", "")

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
