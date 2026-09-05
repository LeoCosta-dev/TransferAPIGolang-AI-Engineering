package httpapi

import (
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type createAccountRequest struct {
	Name     *string `json:"name"`
	Document *string `json:"document"`
}
type updateAccountRequest struct {
	Name *string `json:"name"`
}
type changeStatusRequest struct {
	Status *string `json:"status"`
}
type accountResponse struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Document  string        `json:"document"`
	Status    domain.Status `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
type errorResponse struct {
	Error string `json:"error"`
}
type healthResponse struct {
	Status string `json:"status"`
}

func newAccountResponse(account domain.Account) accountResponse {
	return accountResponse{ID: account.ID, Name: account.Name, Document: account.Document, Status: account.Status, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt}
}
