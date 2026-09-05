package application

import (
	"context"
	"errors"
	"testing"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type repositoryFake struct {
	accounts       map[string]domain.Account
	create, update int
	err            error
}
type financialFake struct{}

func (financialFake) Register(context.Context, string, domain.Status) error { return nil }
func (financialFake) ChangeStatus(context.Context, string, domain.Status, domain.Status) error {
	return nil
}

func (r *repositoryFake) Create(_ context.Context, a domain.Account) error {
	r.create++
	if r.err != nil {
		return r.err
	}
	r.accounts[a.ID] = a
	return nil
}
func (r *repositoryFake) FindByID(_ context.Context, id string) (domain.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return domain.Account{}, ErrAccountNotFound
	}
	return a, nil
}
func (r *repositoryFake) Update(_ context.Context, a domain.Account) error {
	r.update++
	if r.err != nil {
		return r.err
	}
	r.accounts[a.ID] = a
	return nil
}
func TestServiceManagesOnlyAccountData(t *testing.T) {
	r := &repositoryFake{accounts: map[string]domain.Account{}}
	s := NewService(r, financialFake{})
	a, err := s.CreateAccount(context.Background(), "Nome", "Doc")
	if err != nil {
		t.Fatal(err)
	}
	if r.create != 1 {
		t.Fatal("conta não persistida")
	}
	a, err = s.UpdateName(context.Background(), a.ID, "Novo")
	if err != nil || a.Name != "Novo" {
		t.Fatalf("%+v %v", a, err)
	}
	a, err = s.ChangeStatus(context.Background(), a.ID, domain.StatusBlocked)
	if err != nil || a.Status != domain.StatusBlocked {
		t.Fatalf("%+v %v", a, err)
	}
}
func TestServiceDoesNotPersistInvalidInput(t *testing.T) {
	r := &repositoryFake{accounts: map[string]domain.Account{}}
	s := NewService(r, financialFake{})
	if _, err := s.GetAccount(context.Background(), "invalid"); !errors.Is(err, domain.ErrInvalidAccountID) {
		t.Fatal(err)
	}
}
