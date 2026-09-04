package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
	_ "modernc.org/sqlite"
)

func newTestRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	repository, err := New(context.Background(), db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return repository, db
}

func testAccount(t *testing.T) domain.Account {
	t.Helper()
	account, err := domain.NewAccountAt(
		"Nome",
		"document-"+t.Name(),
		time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return account
}

func TestRepositoryPersistsAndReadsAccount(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)

	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found != account {
		t.Fatalf("conta lida = %+v, want %+v", found, account)
	}
}

func TestRepositoryRejectsDuplicateDocument(t *testing.T) {
	repository, _ := newTestRepository(t)
	first := testAccount(t)
	second, err := domain.NewAccount("Outro", first.Document)
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), second); !errors.Is(err, ErrDuplicateDocument) {
		t.Fatalf("erro = %v, want %v", err, ErrDuplicateDocument)
	}
}

func TestRepositoryMapsMissingAccount(t *testing.T) {
	repository, _ := newTestRepository(t)
	_, err := repository.FindByID(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("erro = %v, want %v", err, ErrAccountNotFound)
	}
}

func TestRepositoryUpdatePersistsDomainState(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if err := account.UpdateName("Novo Nome", account.UpdatedAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := account.Credit(100, account.UpdatedAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found != account {
		t.Fatalf("estado atualizado = %+v, want %+v", found, account)
	}
}

func TestRepositoryEnforcesStorageConstraints(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}

	invalid := account
	invalid.Balance = -1
	if err := repository.Update(context.Background(), invalid); !errors.Is(err, ErrStorage) {
		t.Fatalf("saldo negativo: erro = %v, want ErrStorage", err)
	}
	invalid = account
	invalid.Status = domain.Status("INVALID")
	if err := repository.Update(context.Background(), invalid); !errors.Is(err, ErrStorage) {
		t.Fatalf("status inválido: erro = %v, want ErrStorage", err)
	}
}

func TestTransactionRollsBackWhenOperationFails(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}

	err := repository.WithTransaction(context.Background(), func(ctx context.Context, transaction application.AccountTransaction) error {
		loaded, err := transaction.FindByID(ctx, account.ID)
		if err != nil {
			return err
		}
		loaded.Name = "não persistir"
		if err := transaction.Update(ctx, loaded); err != nil {
			return err
		}
		return errors.New("falha simulada")
	})
	if err == nil || err.Error() != "falha simulada" {
		t.Fatalf("erro = %v, want falha simulada", err)
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found != account {
		t.Fatalf("rollback não restaurou conta: %+v", found)
	}
}

func TestConcurrentCreditsDoNotLoseUpdates(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)

	const operations = 20
	errorsChannel := make(chan error, operations)
	var waitGroup sync.WaitGroup
	for index := 0; index < operations; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := service.Credit(context.Background(), account.ID, 1)
			errorsChannel <- err
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("crédito concorrente: %v", err)
		}
	}

	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != operations {
		t.Fatalf("saldo final = %d, want %d", found.Balance, operations)
	}
}
