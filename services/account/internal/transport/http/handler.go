package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type Handler struct{ service application.AccountService }

func NewHandler(service application.AccountService) *Handler { return &Handler{service: service} }

func (handler *Handler) CreateAccount(c echo.Context) error {
	var request createAccountRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}
	if request.Name == nil || request.Document == nil {
		return writeError(c, newRequestError("campos obrigatórios ausentes"))
	}
	account, err := handler.service.CreateAccount(c.Request().Context(), *request.Name, *request.Document)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusCreated, newAccountResponse(account))
}

func (handler *Handler) GetAccount(c echo.Context) error {
	account, err := handler.service.GetAccount(c.Request().Context(), c.Param("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, newAccountResponse(account))
}

func (handler *Handler) UpdateAccount(c echo.Context) error {
	var request updateAccountRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}
	if request.Name == nil {
		return writeError(c, newRequestError("campo name obrigatório"))
	}
	account, err := handler.service.UpdateName(c.Request().Context(), c.Param("id"), *request.Name)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, newAccountResponse(account))
}

func (handler *Handler) ChangeAccountStatus(c echo.Context) error {
	var request changeStatusRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}
	if request.Status == nil || !domain.Status(*request.Status).IsValid() {
		return writeError(c, domain.ErrInvalidStatus)
	}
	account, err := handler.service.ChangeStatus(c.Request().Context(), c.Param("id"), domain.Status(*request.Status))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, newAccountResponse(account))
}

func (handler *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

func decodeJSON(c echo.Context, destination any) error {
	if !strings.HasPrefix(strings.ToLower(c.Request().Header.Get("Content-Type")), "application/json") {
		return newRequestError("Content-Type deve ser application/json")
	}
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return newRequestError("JSON inválido")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return newRequestError("JSON deve conter apenas um objeto")
	}
	return nil
}
