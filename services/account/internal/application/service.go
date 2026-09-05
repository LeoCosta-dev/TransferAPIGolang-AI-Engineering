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
}
type FinancialGateway interface {
	Register(ctx context.Context, id string, status domain.Status) error
	ChangeStatus(ctx context.Context, id string, from, to domain.Status) error
}

type AccountService interface {
	CreateAccount(ctx context.Context, name, document string) (domain.Account, error)
	GetAccount(ctx context.Context, id string) (domain.Account, error)
	UpdateName(ctx context.Context, id, name string) (domain.Account, error)
	ChangeStatus(ctx context.Context, id string, status domain.Status) (domain.Account, error)
}

type Service struct {
	repository AccountRepository
	financial  FinancialGateway
}

var _ AccountService = (*Service)(nil)

func NewService(repository AccountRepository, financial FinancialGateway) *Service {
	return &Service{repository: repository, financial: financial}
}

func (service *Service) CreateAccount(ctx context.Context, name, document string) (domain.Account, error) {
	account, err := domain.NewAccount(name, document)
	if err != nil {
		return domain.Account{}, err
	}
	if err := service.repository.Create(ctx, account); err != nil {
		return domain.Account{}, err
	}
	if err := service.financial.Register(ctx, account.ID, account.Status); err != nil {
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
	previousStatus := account.Status
	if err := account.ChangeStatus(status, time.Now().UTC()); err != nil {
		return domain.Account{}, err
	}
	if err := service.financial.ChangeStatus(ctx, account.ID, previousStatus, status); err != nil {
		return domain.Account{}, err
	}
	if err := service.repository.Update(ctx, account); err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (service *Service) findAccount(ctx context.Context, id string) (domain.Account, error) {
	if err := domain.ValidateAccountID(id); err != nil {
		return domain.Account{}, err
	}
	return service.repository.FindByID(ctx, id)
}
