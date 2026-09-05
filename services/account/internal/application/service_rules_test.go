package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

const stubAccountID = "550e8400-e29b-41d4-a716-446655440000"

type stubAccountRepository struct {
	stored      map[string]domain.Account
	createErr   error
	findErr     error
	updateErr   error
	created     int
	updated     int
	finds       int
	lastUpdated domain.Account
}

func (r *stubAccountRepository) Create(_ context.Context, account domain.Account) error {
	r.created++
	if r.createErr != nil {
		return r.createErr
	}
	if r.stored == nil {
		r.stored = make(map[string]domain.Account)
	}
	r.stored[account.ID] = account
	return nil
}

func (r *stubAccountRepository) FindByID(_ context.Context, id string) (domain.Account, error) {
	r.finds++
	if r.findErr != nil {
		return domain.Account{}, r.findErr
	}
	account, ok := r.stored[id]
	if !ok {
		return domain.Account{}, ErrAccountNotFound
	}
	return account, nil
}

func (r *stubAccountRepository) Update(_ context.Context, account domain.Account) error {
	r.updated++
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.stored == nil {
		r.stored = make(map[string]domain.Account)
	}
	r.stored[account.ID] = account
	r.lastUpdated = account
	return nil
}

type stubFinancialGateway struct {
	registerErr     error
	changeStatusErr error
	registered      int
	changed         int

	lastRegisterID     string
	lastRegisterStatus domain.Status

	lastChangeID                 string
	lastChangeFrom, lastChangeTo domain.Status
}

func (g *stubFinancialGateway) Register(_ context.Context, id string, status domain.Status) error {
	g.registered++
	g.lastRegisterID, g.lastRegisterStatus = id, status
	return g.registerErr
}

func (g *stubFinancialGateway) ChangeStatus(_ context.Context, id string, from, to domain.Status) error {
	g.changed++
	g.lastChangeID, g.lastChangeFrom, g.lastChangeTo = id, from, to
	return g.changeStatusErr
}

func TestCreateAccountSynchronizesFinancialService(t *testing.T) {
	repository := &stubAccountRepository{}
	gateway := &stubFinancialGateway{}
	service := NewService(repository, gateway)

	account, err := service.CreateAccount(context.Background(), " Nome ", "123")
	if err != nil {
		t.Fatal(err)
	}
	if repository.created != 1 {
		t.Fatalf("create = %d, want 1", repository.created)
	}
	if gateway.registered != 1 {
		t.Fatalf("register = %d, want 1", gateway.registered)
	}
	if gateway.lastRegisterID != account.ID {
		t.Fatalf("register id = %q, want %q", gateway.lastRegisterID, account.ID)
	}
	if gateway.lastRegisterStatus != domain.StatusActive {
		t.Fatalf("register status = %q, want %q", gateway.lastRegisterStatus, domain.StatusActive)
	}
}

func TestCreateAccountDoesNotSynchronizeWhenRepositoryFails(t *testing.T) {
	repository := &stubAccountRepository{createErr: ErrDuplicateDocument}
	gateway := &stubFinancialGateway{}
	service := NewService(repository, gateway)

	if _, err := service.CreateAccount(context.Background(), "Nome", "123"); !errors.Is(err, ErrDuplicateDocument) {
		t.Fatalf("erro = %v", err)
	}
	if gateway.registered != 0 {
		t.Fatal("gateway chamado após falha de persistência")
	}
}

// A conta permanece persistida quando o serviço financeiro falha: o serviço
// não possui compensação. O teste protege a propagação do erro; a lacuna de
// consistência está registrada na auditoria.
func TestCreateAccountPropagatesFinancialFailure(t *testing.T) {
	repository := &stubAccountRepository{}
	gateway := &stubFinancialGateway{registerErr: errors.New("transactions indisponível")}
	service := NewService(repository, gateway)

	if _, err := service.CreateAccount(context.Background(), "Nome", "123"); err == nil {
		t.Fatal("esperado erro do serviço financeiro")
	}
}

func TestGetAccountPropagatesRepositoryError(t *testing.T) {
	repository := &stubAccountRepository{findErr: ErrAccountNotFound}
	service := NewService(repository, &stubFinancialGateway{})

	if _, err := service.GetAccount(context.Background(), stubAccountID); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("erro = %v", err)
	}
}

func TestGetAccountValidatesIdentifierBeforePersistence(t *testing.T) {
	repository := &stubAccountRepository{}
	service := NewService(repository, &stubFinancialGateway{})

	if _, err := service.GetAccount(context.Background(), "nao-e-uuid"); !errors.Is(err, domain.ErrInvalidAccountID) {
		t.Fatalf("erro = %v", err)
	}
	if repository.finds != 0 {
		t.Fatalf("finds = %d, want 0", repository.finds)
	}
}

func TestUpdateNamePersistsNormalizedName(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{stored: map[string]domain.Account{
		stubAccountID: {ID: stubAccountID, Name: "Antigo", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now},
	}}
	service := NewService(repository, &stubFinancialGateway{})

	account, err := service.UpdateName(context.Background(), stubAccountID, "  Novo Nome  ")
	if err != nil {
		t.Fatal(err)
	}
	if account.Name != "Novo Nome" {
		t.Fatalf("name = %q, want %q", account.Name, "Novo Nome")
	}
	if repository.updated != 1 {
		t.Fatalf("update = %d, want 1", repository.updated)
	}
	if repository.lastUpdated.Name != "Novo Nome" {
		t.Fatalf("persistido = %q, want %q", repository.lastUpdated.Name, "Novo Nome")
	}
}

func TestUpdateNameRejectsInvalidName(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{stored: map[string]domain.Account{
		stubAccountID: {ID: stubAccountID, Name: "Antigo", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now},
	}}
	service := NewService(repository, &stubFinancialGateway{})

	if _, err := service.UpdateName(context.Background(), stubAccountID, "   "); !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("erro = %v", err)
	}
	if repository.updated != 0 {
		t.Fatalf("update = %d, want 0", repository.updated)
	}
}

func TestUpdateNamePropagatesRepositoryError(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{
		stored:    map[string]domain.Account{stubAccountID: {ID: stubAccountID, Name: "Antigo", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}},
		updateErr: errors.New("mongo indisponível"),
	}
	service := NewService(repository, &stubFinancialGateway{})

	if _, err := service.UpdateName(context.Background(), stubAccountID, "Novo"); err == nil {
		t.Fatal("esperado erro do repositório")
	}
}

func TestUpdateNameValidatesIdentifier(t *testing.T) {
	repository := &stubAccountRepository{}
	service := NewService(repository, &stubFinancialGateway{})

	if _, err := service.UpdateName(context.Background(), "nao-e-uuid", "Novo"); !errors.Is(err, domain.ErrInvalidAccountID) {
		t.Fatalf("erro = %v", err)
	}
	if repository.finds != 0 || repository.updated != 0 {
		t.Fatalf("finds = %d, update = %d, want 0/0", repository.finds, repository.updated)
	}
}

func TestChangeStatusSynchronizesFinancialService(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{stored: map[string]domain.Account{
		stubAccountID: {ID: stubAccountID, Name: "Nome", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now},
	}}
	gateway := &stubFinancialGateway{}
	service := NewService(repository, gateway)

	account, err := service.ChangeStatus(context.Background(), stubAccountID, domain.StatusBlocked)
	if err != nil {
		t.Fatal(err)
	}
	if account.Status != domain.StatusBlocked {
		t.Fatalf("status = %q, want %q", account.Status, domain.StatusBlocked)
	}
	if gateway.changed != 1 {
		t.Fatalf("changed = %d, want 1", gateway.changed)
	}
	if gateway.lastChangeID != stubAccountID || gateway.lastChangeFrom != domain.StatusActive || gateway.lastChangeTo != domain.StatusBlocked {
		t.Fatalf("transição = %q %q→%q", gateway.lastChangeID, gateway.lastChangeFrom, gateway.lastChangeTo)
	}
	if repository.updated != 1 || repository.lastUpdated.Status != domain.StatusBlocked {
		t.Fatalf("update = %d, status = %q", repository.updated, repository.lastUpdated.Status)
	}
}

func TestChangeStatusDoesNotPersistWhenSynchronizationFails(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{stored: map[string]domain.Account{
		stubAccountID: {ID: stubAccountID, Name: "Nome", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now},
	}}
	gateway := &stubFinancialGateway{changeStatusErr: errors.New("transactions indisponível")}
	service := NewService(repository, gateway)

	if _, err := service.ChangeStatus(context.Background(), stubAccountID, domain.StatusBlocked); err == nil {
		t.Fatal("esperado erro do serviço financeiro")
	}
	if repository.updated != 0 {
		t.Fatalf("update = %d, want 0", repository.updated)
	}
}

func TestChangeStatusRejectsInvalidTransition(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	repository := &stubAccountRepository{stored: map[string]domain.Account{
		stubAccountID: {ID: stubAccountID, Name: "Nome", Status: domain.StatusClosed, CreatedAt: now, UpdatedAt: now},
	}}
	gateway := &stubFinancialGateway{}
	service := NewService(repository, gateway)

	if _, err := service.ChangeStatus(context.Background(), stubAccountID, domain.StatusActive); !errors.Is(err, domain.ErrInvalidStatusChange) {
		t.Fatalf("erro = %v", err)
	}
	if gateway.changed != 0 || repository.updated != 0 {
		t.Fatalf("changed = %d, update = %d, want 0/0", gateway.changed, repository.updated)
	}
}

func TestChangeStatusValidatesIdentifier(t *testing.T) {
	repository := &stubAccountRepository{}
	gateway := &stubFinancialGateway{}
	service := NewService(repository, gateway)

	if _, err := service.ChangeStatus(context.Background(), "nao-e-uuid", domain.StatusBlocked); !errors.Is(err, domain.ErrInvalidAccountID) {
		t.Fatalf("erro = %v", err)
	}
	if repository.finds != 0 || gateway.changed != 0 || repository.updated != 0 {
		t.Fatal("persistência ou gateway chamados com identificador inválido")
	}
}
