package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type serviceFake struct{ account domain.Account }

func (fake *serviceFake) CreateAccount(_ context.Context, name, document string) (domain.Account, error) {
	fake.account.Name, fake.account.Document = name, document
	return fake.account, nil
}
func (fake *serviceFake) GetAccount(context.Context, string) (domain.Account, error) {
	return fake.account, nil
}
func (fake *serviceFake) UpdateName(_ context.Context, _, name string) (domain.Account, error) {
	fake.account.Name = name
	return fake.account, nil
}
func (fake *serviceFake) ChangeStatus(_ context.Context, _ string, status domain.Status) (domain.Account, error) {
	fake.account.Status = status
	return fake.account, nil
}
func newServer(service application.AccountService) *echo.Echo {
	e := echo.New()
	RegisterRoutes(e, NewHandler(service))
	return e
}
func TestAccountHTTP(t *testing.T) {
	now := time.Now().UTC()
	fake := &serviceFake{account: domain.Account{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Nome", Document: "123", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}}
	e := newServer(fake)
	for _, test := range []struct {
		method, path, body string
		want               int
	}{{http.MethodGet, "/health", "", 200}, {http.MethodPost, "/api/v1/accounts", `{"name":"Nome","document":"123"}`, 201}, {http.MethodGet, "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000", "", 200}, {http.MethodPatch, "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000", `{"name":"Novo"}`, 200}, {http.MethodPatch, "/api/v1/accounts/550e8400-e29b-41d4-a716-446655440000/status", `{"status":"BLOCKED"}`, 200}} {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		if test.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != test.want {
			t.Errorf("%s %s: status=%d want=%d", test.method, test.path, rec.Code, test.want)
		}
	}
}
