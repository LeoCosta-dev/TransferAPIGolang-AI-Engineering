package application

import (
	"context"
	"errors"
	"strings"

	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
)

const maxIdempotencyKeyLength = 255

var (
	ErrAccountNotFound       = errors.New("conta não encontrada")
	ErrIdempotencyConflict   = errors.New("conflito de idempotência")
	ErrInvalidIdempotencyKey = errors.New("Idempotency-Key inválido")
)

type Repository interface {
	Register(ctx context.Context, id string, status domain.Status) error
	ChangeStatus(ctx context.Context, id string, from, to domain.Status) error
	Apply(ctx context.Context, id string, operation domain.Transaction) (domain.Transaction, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Register(ctx context.Context, id string, status domain.Status) error {
	return s.repository.Register(ctx, id, status)
}
func (s *Service) ChangeStatus(ctx context.Context, id string, from, to domain.Status) error {
	return s.repository.ChangeStatus(ctx, id, from, to)
}

func (s *Service) Credit(ctx context.Context, accountID string, amount int64, key string) (domain.Transaction, error) {
	return s.apply(ctx, accountID, amount, key, domain.TypeCredit)
}
func (s *Service) Debit(ctx context.Context, accountID string, amount int64, key string) (domain.Transaction, error) {
	return s.apply(ctx, accountID, amount, key, domain.TypeDebit)
}
func (s *Service) Balance(ctx context.Context, accountID string) (int64, error) {
	result, err := s.repository.Apply(ctx, accountID, domain.Transaction{Type: "BALANCE"})
	return result.Balance, err
}

func (s *Service) apply(ctx context.Context, accountID string, amount int64, key string, kind domain.Type) (domain.Transaction, error) {
	key = strings.TrimSpace(key)
	if key == "" || len([]rune(key)) > maxIdempotencyKeyLength {
		return domain.Transaction{}, ErrInvalidIdempotencyKey
	}
	return s.repository.Apply(ctx, accountID, domain.Transaction{AccountID: accountID, Type: kind, Amount: amount, IdempotencyKey: key})
}
