package domain

import "errors"

var (
	ErrInvalidName         = errors.New("nome inválido")
	ErrInvalidDocument     = errors.New("documento inválido")
	ErrInvalidAccountID    = errors.New("identificador de conta inválido")
	ErrInvalidStatus       = errors.New("status inválido")
	ErrInvalidStatusChange = errors.New("transição de status inválida")
	ErrAccountBlocked      = errors.New("conta bloqueada")
	ErrAccountClosed       = errors.New("conta fechada")
	ErrInvalidAmount       = errors.New("valor monetário inválido")
	ErrInsufficientBalance = errors.New("saldo insuficiente")
)
