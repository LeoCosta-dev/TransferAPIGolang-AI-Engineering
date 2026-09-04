package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

var errRepository = errors.New("erro do repositório")

type accountRepositoryFake struct {
	accounts    map[string]domain.Account
	createCalls int
	findCalls   int
	updateCalls int
	updateError error
}

func (fake *accountRepositoryFake) WithTransaction(ctx context.Context, operation func(context.Context, AccountTransaction) error) error {
	return operation(ctx, fake)
}

func newAccountRepositoryFake(account domain.Account) *accountRepositoryFake {
	return &accountRepositoryFake{accounts: map[string]domain.Account{account.ID: account}}
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

	credited, err := service.Credit(context.Background(), account.ID, 100)
	if err != nil || credited.Balance != 100 {
		t.Fatalf("Credit() = %+v, %v", credited, err)
	}
	debited, err := service.Debit(context.Background(), account.ID, 40)
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

	_, err := service.Debit(context.Background(), account.ID, 1)
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

	_, err := service.Credit(context.Background(), account.ID, 100)
	if !errors.Is(err, errRepository) {
		t.Fatalf("erro = %v, want %v", err, errRepository)
	}
	if fake.updateCalls != 1 || fake.accounts[account.ID] != account {
		t.Fatal("estado do fake foi alterado apesar do erro de persistência")
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
