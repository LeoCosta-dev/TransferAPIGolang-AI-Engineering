package application

import (
	"context"
	"errors"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
	"testing"
)

type repositoryFake struct {
	operation domain.Transaction
	calls     int
}

func (r *repositoryFake) Register(context.Context, string, domain.Status) error { return nil }
func (r *repositoryFake) ChangeStatus(context.Context, string, domain.Status, domain.Status) error {
	return nil
}
func (r *repositoryFake) Apply(_ context.Context, _ string, o domain.Transaction) (domain.Transaction, error) {
	r.calls++
	r.operation = o
	return domain.Transaction{AccountID: o.AccountID, Type: o.Type, Amount: o.Amount, Balance: 10}, nil
}
func TestServiceValidatesIdempotencyBeforeExternalCalls(t *testing.T) {
	r := &repositoryFake{}
	s := NewService(r)
	if _, err := s.Credit(context.Background(), "id", 1, " "); !errors.Is(err, ErrInvalidIdempotencyKey) {
		t.Fatal(err)
	}
	if r.calls != 0 {
		t.Fatal("persistência chamada")
	}
}
func TestServiceDelegatesFinancialOperation(t *testing.T) {
	r := &repositoryFake{}
	s := NewService(r)
	result, err := s.Debit(context.Background(), "id", 5, "key")
	if err != nil || result.Balance != 10 || r.operation.Type != domain.TypeDebit {
		t.Fatalf("%+v %+v %v", result, r.operation, err)
	}
}
