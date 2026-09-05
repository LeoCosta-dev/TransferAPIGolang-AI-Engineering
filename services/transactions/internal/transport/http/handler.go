package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
)

type service interface {
	Register(context.Context, string, domain.Status) error
	ChangeStatus(context.Context, string, domain.Status, domain.Status) error
	Credit(context.Context, string, int64, string) (domain.Transaction, error)
	Debit(context.Context, string, int64, string) (domain.Transaction, error)
	Balance(context.Context, string) (int64, error)
}
type Handler struct{ service service }

func NewHandler(service service) *Handler { return &Handler{service: service} }

type moneyRequest struct {
	Amount *int64 `json:"amount"`
}
type response struct {
	ID        string      `json:"id,omitempty"`
	AccountID string      `json:"account_id"`
	Type      domain.Type `json:"type,omitempty"`
	Amount    int64       `json:"amount,omitempty"`
	Balance   int64       `json:"balance"`
}

func (h *Handler) Credit(c echo.Context) error { return h.move(c, h.service.Credit) }
func (h *Handler) Debit(c echo.Context) error  { return h.move(c, h.service.Debit) }
func (h *Handler) Balance(c echo.Context) error {
	balance, err := h.service.Balance(c.Request().Context(), c.Param("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, response{AccountID: c.Param("id"), Balance: balance})
}
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) Register(c echo.Context) error {
	var r struct {
		Status domain.Status `json:"status"`
	}
	if err := decodeJSON(c, &r); err != nil {
		return writeError(c, err)
	}
	if !r.Status.IsValid() {
		return writeError(c, requestError{"status inválido"})
	}
	if err := h.service.Register(c.Request().Context(), c.Param("id"), r.Status); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusCreated)
}
func (h *Handler) ChangeStatus(c echo.Context) error {
	var r struct {
		From   domain.Status `json:"from"`
		Status domain.Status `json:"status"`
	}
	if err := decodeJSON(c, &r); err != nil {
		return writeError(c, err)
	}
	if !r.From.IsValid() || !r.Status.IsValid() {
		return writeError(c, requestError{"status inválido"})
	}
	if err := h.service.ChangeStatus(c.Request().Context(), c.Param("id"), r.From, r.Status); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusOK)
}
func (h *Handler) move(c echo.Context, operation func(context.Context, string, int64, string) (domain.Transaction, error)) error {
	var request moneyRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}
	if request.Amount == nil {
		return writeError(c, requestError{"campo amount obrigatório"})
	}
	result, err := operation(c.Request().Context(), c.Param("id"), *request.Amount, c.Request().Header.Get("Idempotency-Key"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, response{ID: result.ID, AccountID: result.AccountID, Type: result.Type, Amount: result.Amount, Balance: result.Balance})
}

type requestError struct{ message string }

func (e requestError) Error() string { return e.message }
func decodeJSON(c echo.Context, destination any) error {
	if !strings.HasPrefix(strings.ToLower(c.Request().Header.Get("Content-Type")), "application/json") {
		return requestError{"Content-Type deve ser application/json"}
	}
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return requestError{"JSON inválido"}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return requestError{"JSON deve conter apenas um objeto"}
	}
	return nil
}
func writeError(c echo.Context, err error) error {
	status := http.StatusInternalServerError
	message := "erro interno do servidor"
	switch {
	case errors.As(err, new(requestError)), errors.Is(err, application.ErrInvalidIdempotencyKey), errors.Is(err, domain.ErrInvalidAmount):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, application.ErrAccountNotFound):
		status = http.StatusNotFound
		message = "conta não encontrada"
	case errors.Is(err, application.ErrIdempotencyConflict), errors.Is(err, domain.ErrAccountBlocked), errors.Is(err, domain.ErrAccountClosed), errors.Is(err, domain.ErrInsufficientBalance), errors.Is(err, domain.ErrInvalidStatusChange):
		status = http.StatusConflict
		message = err.Error()
	}
	return c.JSON(status, map[string]string{"error": message})
}
