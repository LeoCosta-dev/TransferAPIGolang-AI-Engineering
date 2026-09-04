package sqlite

import "errors"

var (
	ErrAccountNotFound   = errors.New("conta não encontrada")
	ErrDuplicateDocument = errors.New("documento já associado a uma conta")
	ErrStorage           = errors.New("erro de persistência")
)
