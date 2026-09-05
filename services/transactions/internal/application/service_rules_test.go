package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
)

type stubRepository struct {
	registerErr     error
	changeStatusErr error
	applyErr        error
	applyResult     domain.Transaction
	applies         []domain.Transaction
	registered      int
	changed         int

	lastRegisterID     string
	lastRegisterStatus domain.Status

	lastChangeID                 string
	lastChangeFrom, lastChangeTo domain.Status
}

func (r *stubRepository) Register(_ context.Context, id string, status domain.Status) error {
	r.registered++
	r.lastRegisterID, r.lastRegisterStatus = id, status
	return r.registerErr
}

func (r *stubRepository) ChangeStatus(_ context.Context, id string, from, to domain.Status) error {
	r.changed++
	r.lastChangeID, r.lastChangeFrom, r.lastChangeTo = id, from, to
	return r.changeStatusErr
}

func (r *stubRepository) Apply(_ context.Context, _ string, operation domain.Transaction) (domain.Transaction, error) {
	r.applies = append(r.applies, operation)
	if r.applyErr != nil {
		return domain.Transaction{}, r.applyErr
	}
	return r.applyResult, nil
}

func TestServiceCreditDelegatesToRepository(t *testing.T) {
	repository := &stubRepository{applyResult: domain.Transaction{
		ID: "acc-1:key", AccountID: "acc-1", Type: domain.TypeCredit, Amount: 500, Balance: 500,
	}}
	service := NewService(repository)

	result, err := service.Credit(context.Background(), "acc-1", 500, "key")
	if err != nil {
		t.Fatal(err)
	}
	if len(repository.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(repository.applies))
	}
	operation := repository.applies[0]
	if operation.AccountID != "acc-1" || operation.Type != domain.TypeCredit || operation.Amount != 500 || operation.IdempotencyKey != "key" {
		t.Fatalf("operação = %+v", operation)
	}
	if result != repository.applyResult {
		t.Fatalf("resultado = %+v, want %+v", result, repository.applyResult)
	}
}

func TestServiceDebitDelegatesToRepository(t *testing.T) {
	repository := &stubRepository{applyResult: domain.Transaction{
		ID: "acc-1:key", AccountID: "acc-1", Type: domain.TypeDebit, Amount: 100, Balance: 0,
	}}
	service := NewService(repository)

	result, err := service.Debit(context.Background(), "acc-1", 100, "key")
	if err != nil {
		t.Fatal(err)
	}
	operation := repository.applies[0]
	if operation.Type != domain.TypeDebit || operation.Amount != 100 {
		t.Fatalf("operação = %+v", operation)
	}
	if result.Balance != 0 {
		t.Fatalf("resultado = %+v", result)
	}
}

// Documenta o contrato atual: a consulta de saldo passa pelo Apply com uma
// operação de tipo "BALANCE" sem AccountID no payload.
func TestServiceBalanceDelegatesToRepository(t *testing.T) {
	repository := &stubRepository{applyResult: domain.Transaction{AccountID: "acc-1", Balance: 77}}
	service := NewService(repository)

	balance, err := service.Balance(context.Background(), "acc-1")
	if err != nil {
		t.Fatal(err)
	}
	if balance != 77 {
		t.Fatalf("balance = %d, want 77", balance)
	}
	if len(repository.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(repository.applies))
	}
	if repository.applies[0] != (domain.Transaction{Type: "BALANCE"}) {
		t.Fatalf("operação = %+v, want %+v", repository.applies[0], domain.Transaction{Type: "BALANCE"})
	}
}

func TestServiceBalancePropagatesRepositoryError(t *testing.T) {
	repository := &stubRepository{applyErr: ErrAccountNotFound}
	service := NewService(repository)

	if _, err := service.Balance(context.Background(), "acc-1"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("erro = %v", err)
	}
}

func TestServiceRegisterDelegates(t *testing.T) {
	repository := &stubRepository{}
	service := NewService(repository)

	if err := service.Register(context.Background(), "acc-1", domain.StatusActive); err != nil {
		t.Fatal(err)
	}
	if repository.registered != 1 || repository.lastRegisterID != "acc-1" || repository.lastRegisterStatus != domain.StatusActive {
		t.Fatalf("register = %d %q %q", repository.registered, repository.lastRegisterID, repository.lastRegisterStatus)
	}
}

func TestServiceChangeStatusDelegates(t *testing.T) {
	repository := &stubRepository{changeStatusErr: errors.New("mongo indisponível")}
	service := NewService(repository)

	err := service.ChangeStatus(context.Background(), "acc-1", domain.StatusActive, domain.StatusBlocked)
	if err == nil {
		t.Fatal("esperado erro do repositório")
	}
	if repository.lastChangeID != "acc-1" || repository.lastChangeFrom != domain.StatusActive || repository.lastChangeTo != domain.StatusBlocked {
		t.Fatalf("transição = %q %q→%q", repository.lastChangeID, repository.lastChangeFrom, repository.lastChangeTo)
	}
}

func TestServiceValidatesIdempotencyKeyRules(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"chave vazia", "", true},
		{"chave só de espaços", "   ", true},
		{"chave no limite", strings.Repeat("k", maxIdempotencyKeyLength), false},
		{"chave acima do limite", strings.Repeat("k", maxIdempotencyKeyLength+1), true},
		{"chave multibyte no limite", strings.Repeat("ã", maxIdempotencyKeyLength), false},
		{"chave multibyte acima do limite", strings.Repeat("ã", maxIdempotencyKeyLength+1), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repository := &stubRepository{}
			service := NewService(repository)

			_, err := service.Credit(context.Background(), "acc-1", 100, tc.key)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidIdempotencyKey) {
					t.Fatalf("erro = %v, want ErrInvalidIdempotencyKey", err)
				}
				if len(repository.applies) != 0 {
					t.Fatal("persistência chamada com chave inválida")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServiceTrimsIdempotencyKey(t *testing.T) {
	repository := &stubRepository{applyResult: domain.Transaction{Balance: 10}}
	service := NewService(repository)

	if _, err := service.Credit(context.Background(), "acc-1", 100, "  key-1  "); err != nil {
		t.Fatal(err)
	}
	if repository.applies[0].IdempotencyKey != "key-1" {
		t.Fatalf("key = %q, want %q", repository.applies[0].IdempotencyKey, "key-1")
	}
}

func TestServicePropagatesRepositoryError(t *testing.T) {
	repository := &stubRepository{applyErr: domain.ErrInsufficientBalance}
	service := NewService(repository)

	if _, err := service.Debit(context.Background(), "acc-1", 100, "key"); !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("erro = %v", err)
	}
}
