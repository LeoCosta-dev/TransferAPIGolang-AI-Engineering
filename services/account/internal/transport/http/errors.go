package httpapi

import (
	"errors"
	"net/http"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type requestError struct {
	message string
}

func (err *requestError) Error() string {
	return err.message
}

func newRequestError(message string) error {
	return &requestError{message: message}
}

func statusForError(err error) (int, string) {
	var requestErr *requestError
	if errors.As(err, &requestErr) {
		return http.StatusBadRequest, requestErr.message
	}

	switch {
	case errors.Is(err, domain.ErrInvalidName):
		return http.StatusBadRequest, "nome inválido"
	case errors.Is(err, domain.ErrInvalidDocument):
		return http.StatusBadRequest, "documento inválido"
	case errors.Is(err, domain.ErrInvalidAccountID):
		return http.StatusBadRequest, "identificador de conta inválido"
	case errors.Is(err, domain.ErrInvalidStatus):
		return http.StatusBadRequest, "status inválido"
	case errors.Is(err, domain.ErrInvalidAmount):
		return http.StatusBadRequest, "valor monetário inválido"
	case errors.Is(err, application.ErrAccountNotFound):
		return http.StatusNotFound, "conta não encontrada"
	case errors.Is(err, application.ErrDuplicateDocument):
		return http.StatusConflict, "documento já associado a uma conta"
	case errors.Is(err, application.ErrIdempotencyConflict):
		return http.StatusConflict, "Idempotency-Key em conflito com a operação existente"
	case errors.Is(err, domain.ErrInvalidStatusChange):
		return http.StatusConflict, "transição de status não permitida"
	case errors.Is(err, domain.ErrAccountBlocked):
		return http.StatusConflict, "conta bloqueada"
	case errors.Is(err, domain.ErrAccountClosed):
		return http.StatusConflict, "conta fechada"
	case errors.Is(err, domain.ErrInsufficientBalance):
		return http.StatusConflict, "saldo insuficiente"
	default:
		return http.StatusInternalServerError, "erro interno do servidor"
	}
}

func writeError(c echoContext, err error) error {
	status, message := statusForError(err)
	return c.JSON(status, errorResponse{Error: message})
}

type echoContext interface {
	JSON(code int, i interface{}) error
}
