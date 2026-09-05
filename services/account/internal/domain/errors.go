package domain

import "errors"

var (
	ErrInvalidName         = errors.New("nome inválido")
	ErrInvalidDocument     = errors.New("documento inválido")
	ErrInvalidAccountID    = errors.New("identificador de conta inválido")
	ErrInvalidStatus       = errors.New("status inválido")
	ErrInvalidStatusChange = errors.New("transição de status inválida")
)
