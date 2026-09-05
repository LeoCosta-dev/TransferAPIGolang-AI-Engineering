package httpapi

import (
	"errors"
	"net/http"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type requestError struct{ message string }

func (err *requestError) Error() string    { return err.message }
func newRequestError(message string) error { return &requestError{message: message} }

func statusForError(err error) (int, string) {
	var requestErr *requestError
	if errors.As(err, &requestErr) {
		return http.StatusBadRequest, requestErr.message
	}
	switch {
	case errors.Is(err, domain.ErrInvalidName), errors.Is(err, domain.ErrInvalidDocument), errors.Is(err, domain.ErrInvalidAccountID), errors.Is(err, domain.ErrInvalidStatus):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, application.ErrAccountNotFound):
		return http.StatusNotFound, "conta não encontrada"
	case errors.Is(err, application.ErrDuplicateDocument):
		return http.StatusConflict, "documento já associado a uma conta"
	case errors.Is(err, domain.ErrInvalidStatusChange):
		return http.StatusConflict, "transição de status não permitida"
	case errors.Is(err, application.ErrAccountHasBalance):
		return http.StatusConflict, "conta possui saldo"
	default:
		return http.StatusInternalServerError, "erro interno do servidor"
	}
}

func writeError(c interface{ JSON(int, interface{}) error }, err error) error {
	status, message := statusForError(err)
	return c.JSON(status, errorResponse{Error: message})
}
