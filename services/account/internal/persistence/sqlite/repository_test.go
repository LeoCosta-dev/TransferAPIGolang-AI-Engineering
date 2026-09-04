package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
	_ "modernc.org/sqlite"
)

func newTestRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", sqliteDSN(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())))
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

func TestSQLiteDSNAddsForeignKeysWithoutDroppingParameters(t *testing.T) {
	if got := sqliteDSN("file:account.db"); got != "file:account.db?_pragma=foreign_keys(1)" {
		t.Fatalf("DSN sem parâmetros = %q", got)
	}
	if got := sqliteDSN("file:account.db?mode=rwc"); got != "file:account.db?mode=rwc&_pragma=foreign_keys(1)" {
		t.Fatalf("DSN com parâmetros = %q", got)
	}
}

func TestForeignKeysAreEnabledForPoolConnections(t *testing.T) {
	repository, db := newTestRepository(t)
	if repository == nil {
		t.Fatal("repository não inicializado")
	}
	db.SetMaxOpenConns(8)

	for index := 0; index < 8; index++ {
		_, err := db.ExecContext(context.Background(), `
INSERT INTO idempotency_operations (idempotency_key, account_id, operation_type, amount, resulting_balance, created_at)
VALUES (?, ?, 'CREDIT', 1, 1, ?)`,
			fmt.Sprintf("foreign-key-%d", index),
			"550e8400-e29b-41d4-a716-446655440001",
			formatTime(time.Now()),
		)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			t.Fatalf("tentativa %d: erro = %v, esperava rejeição de foreign key", index, err)
		}
	}
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
			_, err := service.Credit(context.Background(), account.ID, 1, fmt.Sprintf("credit-%d", index))
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

func TestIdempotentCreditAndDebitReplayWithRealSQLite(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)

	credited, err := service.Credit(context.Background(), account.ID, 100, "credit-replay")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Credit(context.Background(), account.ID, 100, "credit-replay")
	if err != nil {
		t.Fatal(err)
	}
	if credited.Balance != 100 || replayed.Balance != credited.Balance {
		t.Fatalf("crédito/replay: first=%d replay=%d", credited.Balance, replayed.Balance)
	}

	debited, err := service.Debit(context.Background(), account.ID, 40, "debit-replay")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err = service.Debit(context.Background(), account.ID, 40, "debit-replay")
	if err != nil {
		t.Fatal(err)
	}
	if debited.Balance != 60 || replayed.Balance != debited.Balance {
		t.Fatalf("débito/replay: first=%d replay=%d", debited.Balance, replayed.Balance)
	}

	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != 60 {
		t.Fatalf("saldo final = %d, want 60", found.Balance)
	}
}

func TestIdempotencyConflictsWithRealSQLite(t *testing.T) {
	repository, _ := newTestRepository(t)
	first := testAccount(t)
	second, err := domain.NewAccount("Outro", "other-"+t.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)
	if _, err := service.Credit(context.Background(), first.ID, 100, "conflict"); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		accountID string
		amount    int64
		debit     bool
	}{
		{"different amount", first.ID, 200, false},
		{"different operation", first.ID, 100, true},
		{"different account", second.ID, 100, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.debit {
				_, err = service.Debit(context.Background(), test.accountID, test.amount, "conflict")
			} else {
				_, err = service.Credit(context.Background(), test.accountID, test.amount, "conflict")
			}
			if !errors.Is(err, application.ErrIdempotencyConflict) {
				t.Fatalf("erro = %v, want %v", err, application.ErrIdempotencyConflict)
			}
		})
	}

	found, err := repository.FindByID(context.Background(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != 100 {
		t.Fatalf("saldo alterado após conflito: %d", found.Balance)
	}
}

func TestFailedBusinessOperationCanReuseKey(t *testing.T) {
	repository, _ := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)

	if _, err := service.Debit(context.Background(), account.ID, 1, "retry"); !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("erro = %v, want %v", err, domain.ErrInsufficientBalance)
	}
	if _, err := service.Credit(context.Background(), account.ID, 1, "retry"); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != 1 {
		t.Fatalf("saldo após retry = %d, want 1", found.Balance)
	}
}

func TestIdempotencyPersistsAfterRepositoryRecreation(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "account.db")
	repository, err := Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		repository.Close()
		t.Fatal(err)
	}
	service := application.NewService(repository)
	if _, err := service.Credit(context.Background(), account.ID, 100, "restart"); err != nil {
		repository.Close()
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	repository, err = Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service = application.NewService(repository)
	replayed, err := service.Credit(context.Background(), account.ID, 100, "restart")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Balance != 100 {
		t.Fatalf("saldo replayado após reinício = %d, want 100", replayed.Balance)
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != 100 {
		t.Fatalf("saldo persistido após reinício = %d, want 100", found.Balance)
	}
}

func TestConcurrentSameIdempotencyKeyHasOneEffect(t *testing.T) {
	repository, db := newTestRepository(t)
	account := testAccount(t)
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)

	const operations = 20
	errorsChannel := make(chan error, operations)
	balances := make(chan int64, operations)
	var waitGroup sync.WaitGroup
	for index := 0; index < operations; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, err := service.Credit(context.Background(), account.ID, 1, "same-key")
			errorsChannel <- err
			if err == nil {
				balances <- result.Balance
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	close(balances)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("operação concorrente: %v", err)
		}
	}
	for balance := range balances {
		if balance != 1 {
			t.Fatalf("resultado concorrente = %d, want 1", balance)
		}
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != 1 {
		t.Fatalf("efeito duplicado: saldo = %d, want 1", found.Balance)
	}
	var records int
	if err := db.QueryRow("SELECT COUNT(*) FROM idempotency_operations WHERE idempotency_key = ?", "same-key").Scan(&records); err != nil {
		t.Fatal(err)
	}
	if records != 1 {
		t.Fatalf("registros de idempotência = %d, want 1", records)
	}
}

func TestTransactionRollsBackAccountAndIdempotencyTogether(t *testing.T) {
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
		loaded.Balance = 100
		if err := transaction.Update(ctx, loaded); err != nil {
			return err
		}
		return transaction.CreateIdempotencyOperation(ctx, application.IdempotencyOperation{
			Key:              "",
			AccountID:        account.ID,
			OperationType:    application.OperationCredit,
			Amount:           100,
			ResultingBalance: 100,
		})
	})
	if err == nil {
		t.Fatal("transação inválida foi confirmada")
	}
	found, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Balance != account.Balance {
		t.Fatalf("saldo após rollback = %d, want %d", found.Balance, account.Balance)
	}
}
