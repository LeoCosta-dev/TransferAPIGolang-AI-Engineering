package httpapi

import "github.com/labstack/echo/v4"

func RegisterRoutes(e *echo.Echo, handler *Handler) {
	e.GET("/health", handler.Health)
	accounts := e.Group("/api/v1/accounts")
	accounts.POST("", handler.CreateAccount)
	accounts.GET("/:id", handler.GetAccount)
	accounts.PATCH("/:id", handler.UpdateAccount)
	accounts.PATCH("/:id/status", handler.ChangeAccountStatus)
}
