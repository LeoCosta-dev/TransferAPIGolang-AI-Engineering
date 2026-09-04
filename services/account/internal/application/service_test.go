package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

var errRepository = errors.New("erro do repositório")

type accountRepositoryFake struct {
	accounts         map[string]domain.Account
	createCalls      int
	findCalls        int
	updateCalls      int
	updateError      error
	operations       map[string]IdempotencyOperation
	transactionCalls int
}

func (fake *accountRepositoryFake) WithTransaction(ctx context.Context, operation func(context.Context, AccountTransaction) error) error {
	fake.transactionCalls++
	return operation(ctx, fake)
}

func (fake *accountRepositoryFake) FindIdempotencyOperation(_ context.Context, key string) (IdempotencyOperation, error) {
	operation, exists := fake.operations[key]
	if !exists {
		return IdempotencyOperation{}, ErrIdempotencyNotFound
	}
	return operation, nil
}

func (fake *accountRepositoryFake) CreateIdempotencyOperation(_ context.Context, operation IdempotencyOperation) error {
	if fake.operations == nil {
		fake.operations = make(map[string]IdempotencyOperation)
	}
	fake.operations[operation.Key] = operation
	return nil
}

func newAccountRepositoryFake(account domain.Account) *accountRepositoryFake {
	return &accountRepositoryFake{accounts: map[string]domain.Account{account.ID: account}, operations: make(map[string]IdempotencyOperation)}
}

func (fake *accountRepositoryFake) Create(_ context.Context, account domain.Account) error {
	fake.createCalls++
	if _, exists := fake.accounts[account.ID]; exists {
		return errRepository
	}
	fake.accounts[account.ID] = account
	return nil
}

func (fake *accountRepositoryFake) FindByID(_ context.Context, id string) (domain.Account, error) {
	fake.findCalls++
	account, exists := fake.accounts[id]
	if !exists {
		return domain.Account{}, domain.ErrInvalidAccountID
	}
	return account, nil
}

func (fake *accountRepositoryFake) Update(_ context.Context, account domain.Account) error {
	fake.updateCalls++
	if fake.updateError != nil {
		return fake.updateError
	}
	fake.accounts[account.ID] = account
	return nil
}

func newTestAccount(t *testing.T) domain.Account {
	t.Helper()
	account, err := domain.NewAccount("Nome", "123")
	if err != nil {
		t.Fatal(err)
	}
	return account
}

func TestCreateAccount(t *testing.T) {
	fake := &accountRepositoryFake{accounts: make(map[string]domain.Account)}
	service := NewService(fake)

	account, err := service.CreateAccount(context.Background(), "  Nome ", " 123 ")
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if account.Name != "Nome" || account.Document != "123" || account.Balance != 0 || account.Status != domain.StatusActive {
		t.Fatalf("conta criada incorretamente: %+v", account)
	}
	if fake.createCalls != 1 || fake.accounts[account.ID] != account {
		t.Fatalf("interação de criação incorreta: calls=%d", fake.createCalls)
	}
}

func TestGetAccountAndBalance(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)

	found, err := service.GetAccount(context.Background(), account.ID)
	if err != nil || found != account {
		t.Fatalf("GetAccount() = %+v, %v", found, err)
	}
	balance, err := service.GetBalance(context.Background(), account.ID)
	if err != nil || balance != account.Balance {
		t.Fatalf("GetBalance() = %d, %v", balance, err)
	}
	if fake.findCalls != 2 || fake.updateCalls != 0 {
		t.Fatalf("interação de consulta incorreta: finds=%d updates=%d", fake.findCalls, fake.updateCalls)
	}
}

func TestInvalidIDDoesNotAccessRepository(t *testing.T) {
	fake := &accountRepositoryFake{accounts: make(map[string]domain.Account)}
	service := NewService(fake)

	_, err := service.GetAccount(context.Background(), "invalid")
	if !errors.Is(err, domain.ErrInvalidAccountID) {
		t.Fatalf("erro = %v, want %v", err, domain.ErrInvalidAccountID)
	}
	if fake.findCalls != 0 {
		t.Fatal("repositório foi consultado para ID inválido")
	}
}

func TestUpdateNameAndChangeStatus(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)

	updated, err := service.UpdateName(context.Background(), account.ID, "Novo Nome")
	if err != nil || updated.Name != "Novo Nome" {
		t.Fatalf("UpdateName() = %+v, %v", updated, err)
	}
	updated, err = service.ChangeStatus(context.Background(), account.ID, domain.StatusBlocked)
	if err != nil || updated.Status != domain.StatusBlocked {
		t.Fatalf("ChangeStatus() = %+v, %v", updated, err)
	}
	if fake.updateCalls != 2 || fake.accounts[account.ID].Status != domain.StatusBlocked {
		t.Fatalf("interação de atualização incorreta: calls=%d", fake.updateCalls)
	}
}

func TestCreditAndDebit(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)

	credited, err := service.Credit(context.Background(), account.ID, 100, "credit-1")
	if err != nil || credited.Balance != 100 {
		t.Fatalf("Credit() = %+v, %v", credited, err)
	}
	debited, err := service.Debit(context.Background(), account.ID, 40, "debit-1")
	if err != nil || debited.Balance != 60 {
		t.Fatalf("Debit() = %+v, %v", debited, err)
	}
	if fake.updateCalls != 2 || fake.accounts[account.ID].Balance != 60 {
		t.Fatalf("saldo persistido incorretamente: %+v", fake.accounts[account.ID])
	}
}

func TestDomainFailureDoesNotUpdate(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)

	_, err := service.Debit(context.Background(), account.ID, 1, "debit-failed")
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("erro = %v, want %v", err, domain.ErrInsufficientBalance)
	}
	if fake.updateCalls != 0 || fake.accounts[account.ID] != account {
		t.Fatal("falha de domínio produziu atualização")
	}

	_, err = service.UpdateName(context.Background(), account.ID, "   ")
	if !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("erro = %v, want %v", err, domain.ErrInvalidName)
	}
	if fake.updateCalls != 0 {
		t.Fatal("falha de validação produziu atualização")
	}
}

func TestRepositoryErrorIsPropagatedWithoutChangingStoredAccount(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	fake.updateError = errRepository
	service := NewService(fake)

	_, err := service.Credit(context.Background(), account.ID, 100, "credit-error")
	if !errors.Is(err, errRepository) {
		t.Fatalf("erro = %v, want %v", err, errRepository)
	}
	if fake.updateCalls != 1 || fake.accounts[account.ID] != account {
		t.Fatal("estado do fake foi alterado apesar do erro de persistência")
	}
}

func TestCreditReplaysStoredResultWithoutUpdatingAccount(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	fake.operations["credit-1"] = IdempotencyOperation{
		Key:              "credit-1",
		AccountID:        account.ID,
		OperationType:    OperationCredit,
		Amount:           100,
		ResultingBalance: 100,
	}
	service := NewService(fake)

	result, err := service.Credit(context.Background(), account.ID, 100, "credit-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountID != account.ID || result.Balance != 100 {
		t.Fatalf("resultado = %+v", result)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", fake.updateCalls)
	}
}

func TestIdempotencyConflictsAreReturnedBeforeDomainMutation(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	fake.operations["existing"] = IdempotencyOperation{
		Key:              "existing",
		AccountID:        account.ID,
		OperationType:    OperationCredit,
		Amount:           100,
		ResultingBalance: 100,
	}
	service := NewService(fake)

	for _, test := range []struct {
		name      string
		accountID string
		operation OperationType
		amount    int64
	}{
		{"different account", "550e8400-e29b-41d4-a716-446655440001", OperationCredit, 100},
		{"different operation", account.ID, OperationDebit, 100},
		{"different amount", account.ID, OperationCredit, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.operation == OperationCredit {
				_, err = service.Credit(context.Background(), test.accountID, test.amount, "existing")
			} else {
				_, err = service.Debit(context.Background(), test.accountID, test.amount, "existing")
			}
			if !errors.Is(err, ErrIdempotencyConflict) {
				t.Fatalf("erro = %v, want %v", err, ErrIdempotencyConflict)
			}
			if fake.updateCalls != 0 {
				t.Fatal("conflito produziu atualização")
			}
		})
	}
}

func TestFailedDomainOperationDoesNotCreateIdempotencyRecord(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)

	_, err := service.Debit(context.Background(), account.ID, 1, "retryable")
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatal(err)
	}
	if _, exists := fake.operations["retryable"]; exists {
		t.Fatal("falha de domínio criou registro de idempotência")
	}
}

func TestInvalidIdempotencyKeyDoesNotStartTransaction(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func(*Service, context.Context, string, int64, string) (MoneyOperationResult, error)
	}{
		{name: "credit", call: (*Service).Credit},
		{name: "debit", call: (*Service).Debit},
	} {
		t.Run(operation.name, func(t *testing.T) {
			account, err := domain.NewAccount("Nome", "123")
			if err != nil {
				t.Fatal(err)
			}
			fake := newAccountRepositoryFake(account)
			service := NewService(fake)

			for _, key := range []string{"", "   ", strings.Repeat("a", maxIdempotencyKeyLength+1)} {
				if _, err := operation.call(service, context.Background(), account.ID, 1, key); !errors.Is(err, ErrInvalidIdempotencyKey) {
					t.Errorf("chave %q: erro = %v, want %v", key, err, ErrInvalidIdempotencyKey)
				}
			}
			if fake.transactionCalls != 0 || fake.updateCalls != 0 || len(fake.operations) != 0 {
				t.Fatalf("chave inválida acessou persistência: transactions=%d updates=%d operations=%d", fake.transactionCalls, fake.updateCalls, len(fake.operations))
			}
		})
	}
}

func TestServicePassesContextToRepository(t *testing.T) {
	account := newTestAccount(t)
	fake := newAccountRepositoryFake(account)
	service := NewService(fake)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := service.GetAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
}
