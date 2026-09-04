package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account domain.Account) error
	FindByID(ctx context.Context, id string) (domain.Account, error)
	Update(ctx context.Context, account domain.Account) error
	WithTransaction(ctx context.Context, operation func(context.Context, AccountTransaction) error) error
}

type AccountService interface {
	CreateAccount(ctx context.Context, name, document string) (domain.Account, error)
	GetAccount(ctx context.Context, id string) (domain.Account, error)
	UpdateName(ctx context.Context, id, name string) (domain.Account, error)
	ChangeStatus(ctx context.Context, id string, status domain.Status) (domain.Account, error)
	GetBalance(ctx context.Context, id string) (int64, error)
	Credit(ctx context.Context, id string, amount int64, idempotencyKey string) (MoneyOperationResult, error)
	Debit(ctx context.Context, id string, amount int64, idempotencyKey string) (MoneyOperationResult, error)
}

type AccountTransaction interface {
	FindByID(ctx context.Context, id string) (domain.Account, error)
	Update(ctx context.Context, account domain.Account) error
	FindIdempotencyOperation(ctx context.Context, key string) (IdempotencyOperation, error)
	CreateIdempotencyOperation(ctx context.Context, operation IdempotencyOperation) error
}

type Service struct {
	repository AccountRepository
}

var _ AccountService = (*Service)(nil)

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

func (service *Service) Credit(ctx context.Context, id string, amount int64, idempotencyKey string) (MoneyOperationResult, error) {
	var result MoneyOperationResult
	idempotencyKey, err := normalizeIdempotencyKey(idempotencyKey)
	if err != nil {
		return MoneyOperationResult{}, err
	}
	err = service.repository.WithTransaction(ctx, func(transactionContext context.Context, transaction AccountTransaction) error {
		if operation, err := service.findIdempotencyOperation(transactionContext, transaction, idempotencyKey, OperationCredit, id, amount); err == nil {
			result = replayResult(operation)
			return nil
		} else if !errors.Is(err, ErrIdempotencyNotFound) {
			return err
		}

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
		if err := transaction.CreateIdempotencyOperation(transactionContext, IdempotencyOperation{
			Key:              idempotencyKey,
			AccountID:        id,
			OperationType:    OperationCredit,
			Amount:           amount,
			ResultingBalance: account.Balance,
		}); err != nil {
			return err
		}
		result = MoneyOperationResult{AccountID: account.ID, Balance: account.Balance}
		return nil
	})
	if err != nil {
		return MoneyOperationResult{}, err
	}
	return result, nil
}

func (service *Service) Debit(ctx context.Context, id string, amount int64, idempotencyKey string) (MoneyOperationResult, error) {
	var result MoneyOperationResult
	idempotencyKey, err := normalizeIdempotencyKey(idempotencyKey)
	if err != nil {
		return MoneyOperationResult{}, err
	}
	err = service.repository.WithTransaction(ctx, func(transactionContext context.Context, transaction AccountTransaction) error {
		if operation, err := service.findIdempotencyOperation(transactionContext, transaction, idempotencyKey, OperationDebit, id, amount); err == nil {
			result = replayResult(operation)
			return nil
		} else if !errors.Is(err, ErrIdempotencyNotFound) {
			return err
		}

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
		if err := transaction.CreateIdempotencyOperation(transactionContext, IdempotencyOperation{
			Key:              idempotencyKey,
			AccountID:        id,
			OperationType:    OperationDebit,
			Amount:           amount,
			ResultingBalance: account.Balance,
		}); err != nil {
			return err
		}
		result = MoneyOperationResult{AccountID: account.ID, Balance: account.Balance}
		return nil
	})
	if err != nil {
		return MoneyOperationResult{}, err
	}
	return result, nil
}

func (service *Service) findIdempotencyOperation(ctx context.Context, transaction AccountTransaction, key string, operationType OperationType, accountID string, amount int64) (IdempotencyOperation, error) {
	operation, err := transaction.FindIdempotencyOperation(ctx, key)
	if err != nil {
		return IdempotencyOperation{}, err
	}
	if operation.AccountID != accountID || operation.OperationType != operationType || operation.Amount != amount {
		return IdempotencyOperation{}, ErrIdempotencyConflict
	}
	return operation, nil
}

func replayResult(operation IdempotencyOperation) MoneyOperationResult {
	return MoneyOperationResult{AccountID: operation.AccountID, Balance: operation.ResultingBalance}
}

func normalizeIdempotencyKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > maxIdempotencyKeyLength {
		return "", ErrInvalidIdempotencyKey
	}
	return value, nil
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
