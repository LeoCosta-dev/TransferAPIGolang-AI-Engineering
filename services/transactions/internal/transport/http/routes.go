package httpapi

import "github.com/labstack/echo/v4"

func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.GET("/health", h.Health)
	g := e.Group("/api/v1/transactions")
	g.GET("/:id/balance", h.Balance)
	g.POST("/:id/credits", h.Credit)
	g.POST("/:id/debits", h.Debit)
	internal := e.Group("/internal/v1/accounts")
	internal.POST("/:id/register", h.Register)
	internal.POST("/:id/status", h.ChangeStatus)
}
