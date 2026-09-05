package application

import "errors"

var (
	ErrAccountNotFound   = errors.New("conta não encontrada")
	ErrDuplicateDocument = errors.New("documento já associado a uma conta")
	ErrAccountHasBalance = errors.New("conta possui saldo")
)
