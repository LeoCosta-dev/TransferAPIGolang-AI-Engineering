package domain

import (
	"errors"
	"math"
)

type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusBlocked Status = "BLOCKED"
	StatusClosed  Status = "CLOSED"
)

func (status Status) IsValid() bool {
	return status == StatusActive || status == StatusBlocked || status == StatusClosed
}

var (
	ErrInvalidAmount       = errors.New("valor monetário inválido")
	ErrAccountBlocked      = errors.New("conta bloqueada")
	ErrAccountClosed       = errors.New("conta fechada")
	ErrInsufficientBalance = errors.New("saldo insuficiente")
	ErrInvalidStatusChange = errors.New("transição de status inválida")
)

type Account struct {
	ID      string
	Status  Status
	Balance int64
}

func CanTransition(from, to Status) bool {
	return (from == StatusActive && (to == StatusBlocked || to == StatusClosed)) ||
		(from == StatusBlocked && (to == StatusActive || to == StatusClosed))
}

type Type string

const (
	TypeCredit Type = "CREDIT"
	TypeDebit  Type = "DEBIT"
)

type Transaction struct {
	ID             string
	AccountID      string
	Type           Type
	Amount         int64
	Balance        int64
	IdempotencyKey string
}

func (account *Account) Credit(amount int64) error {
	if amount <= 0 || account.Balance > math.MaxInt64-amount {
		return ErrInvalidAmount
	}
	if account.Status == StatusBlocked {
		return ErrAccountBlocked
	}
	if account.Status == StatusClosed {
		return ErrAccountClosed
	}
	if account.Status != StatusActive {
		return ErrAccountBlocked
	}
	account.Balance += amount
	return nil
}
func (account *Account) Debit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if account.Status == StatusBlocked {
		return ErrAccountBlocked
	}
	if account.Status == StatusClosed {
		return ErrAccountClosed
	}
	if account.Status != StatusActive {
		return ErrAccountBlocked
	}
	if amount > account.Balance {
		return ErrInsufficientBalance
	}
	account.Balance -= amount
	return nil
}
