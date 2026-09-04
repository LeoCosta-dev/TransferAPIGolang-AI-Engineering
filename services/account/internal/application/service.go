package application

import (
	"context"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account domain.Account) error
	FindByID(ctx context.Context, id string) (domain.Account, error)
	Update(ctx context.Context, account domain.Account) error
	WithTransaction(ctx context.Context, operation func(context.Context, AccountTransaction) error) error
}

type AccountTransaction interface {
	FindByID(ctx context.Context, id string) (domain.Account, error)
	Update(ctx context.Context, account domain.Account) error
}

type Service struct {
	repository AccountRepository
}

func NewService(repository AccountRepository) *Service {
	return &Service{repository: repository}
}

func (service *Service) CreateAccount(ctx context.Context, name, document string) (domain.Account, error) {
	account, err := domain.NewAccount(name, document)
	if err != nil {
		return domain.Account{}, err
	}
	if err := service.repository.Create(ctx, account); err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (service *Service) GetAccount(ctx context.Context, id string) (domain.Account, error) {
	if err := domain.ValidateAccountID(id); err != nil {
		return domain.Account{}, err
	}
	return service.repository.FindByID(ctx, id)
}

func (service *Service) UpdateName(ctx context.Context, id, name string) (domain.Account, error) {
	account, err := service.findAccount(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}
	if err := account.UpdateName(name, time.Now().UTC()); err != nil {
		return domain.Account{}, err
	}
	if err := service.repository.Update(ctx, account); err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (service *Service) ChangeStatus(ctx context.Context, id string, status domain.Status) (domain.Account, error) {
	account, err := service.findAccount(ctx, id)
	if err != nil {
		return domain.Account{}, err
	}
	if err := account.ChangeStatus(status, time.Now().UTC()); err != nil {
		return domain.Account{}, err
	}
	if err := service.repository.Update(ctx, account); err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (service *Service) GetBalance(ctx context.Context, id string) (int64, error) {
	account, err := service.findAccount(ctx, id)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

func (service *Service) Credit(ctx context.Context, id string, amount int64) (domain.Account, error) {
	var result domain.Account
	err := service.repository.WithTransaction(ctx, func(transactionContext context.Context, transaction AccountTransaction) error {
		account, err := service.findAccountInTransaction(transactionContext, transaction, id)
		if err != nil {
			return err
		}
		if err := account.Credit(amount, time.Now().UTC()); err != nil {
			return err
		}
		if err := transaction.Update(transactionContext, account); err != nil {
			return err
		}
		result = account
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}
	return result, nil
}

func (service *Service) Debit(ctx context.Context, id string, amount int64) (domain.Account, error) {
	var result domain.Account
	err := service.repository.WithTransaction(ctx, func(transactionContext context.Context, transaction AccountTransaction) error {
		account, err := service.findAccountInTransaction(transactionContext, transaction, id)
		if err != nil {
			return err
		}
		if err := account.Debit(amount, time.Now().UTC()); err != nil {
			return err
		}
		if err := transaction.Update(transactionContext, account); err != nil {
			return err
		}
		result = account
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}
	return result, nil
}

func (service *Service) findAccount(ctx context.Context, id string) (domain.Account, error) {
	if err := domain.ValidateAccountID(id); err != nil {
		return domain.Account{}, err
	}
	return service.repository.FindByID(ctx, id)
}

func (service *Service) findAccountInTransaction(ctx context.Context, transaction AccountTransaction, id string) (domain.Account, error) {
	if err := domain.ValidateAccountID(id); err != nil {
		return domain.Account{}, err
	}
	return transaction.FindByID(ctx, id)
}
