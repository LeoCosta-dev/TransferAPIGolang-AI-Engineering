package application

import "errors"

const maxIdempotencyKeyLength = 255

type OperationType string

const (
	OperationCredit OperationType = "CREDIT"
	OperationDebit  OperationType = "DEBIT"
)

type IdempotencyOperation struct {
	Key              string
	AccountID        string
	OperationType    OperationType
	Amount           int64
	ResultingBalance int64
}

type MoneyOperationResult struct {
	AccountID string
	Balance   int64
}

var (
	ErrIdempotencyNotFound   = errors.New("operação idempotente não encontrada")
	ErrIdempotencyConflict   = errors.New("conflito de idempotência")
	ErrInvalidIdempotencyKey = errors.New("Idempotency-Key inválido")
)
