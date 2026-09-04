package sqlite

import (
	"errors"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
)

var (
	ErrAccountNotFound   = application.ErrAccountNotFound
	ErrDuplicateDocument = application.ErrDuplicateDocument
	ErrStorage           = errors.New("erro de persistência")
)
