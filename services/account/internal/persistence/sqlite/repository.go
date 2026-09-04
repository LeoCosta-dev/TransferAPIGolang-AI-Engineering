package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
	_ "modernc.org/sqlite"
)

var _ application.AccountRepository = (*Repository)(nil)

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, dataSourceName string) (*Repository, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("abrir banco SQLite: %w", err)
	}

	repository, err := New(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func New(ctx context.Context, db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("inicializar repository: %w", ErrStorage)
	}

	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return nil, fmt.Errorf("configurar SQLite: %w: %v", ErrStorage, err)
		}
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("criar schema SQLite: %w: %v", ErrStorage, err)
	}
	return &Repository{db: db}, nil
}

func (repository *Repository) Create(ctx context.Context, account domain.Account) error {
	_, err := repository.db.ExecContext(ctx, `
INSERT INTO accounts (id, name, document, balance, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		account.ID,
		account.Name,
		account.Document,
		account.Balance,
		account.Status,
		formatTime(account.CreatedAt),
		formatTime(account.UpdatedAt),
	)
	if err != nil {
		if isDuplicateDocumentError(err) {
			return ErrDuplicateDocument
		}
		return fmt.Errorf("criar conta: %w: %v", ErrStorage, err)
	}
	return nil
}

func (repository *Repository) FindByID(ctx context.Context, id string) (domain.Account, error) {
	return findByID(ctx, repository.db, id)
}

func (repository *Repository) Update(ctx context.Context, account domain.Account) error {
	return update(ctx, repository.db, account)
}

func (repository *Repository) WithTransaction(ctx context.Context, operation func(context.Context, application.AccountTransaction) error) error {
	connection, err := repository.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("abrir conexão SQLite: %w: %v", ErrStorage, err)
	}
	defer connection.Close()

	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("iniciar transação SQLite: %w: %v", ErrStorage, err)
	}

	transaction := &accountTransaction{connection: connection}
	if err := operation(ctx, transaction); err != nil {
		if rollbackErr := rollback(ctx, connection); rollbackErr != nil {
			return fmt.Errorf("%w: falha na operação: %v; rollback: %v", ErrStorage, err, rollbackErr)
		}
		return err
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		_ = rollback(context.Background(), connection)
		return fmt.Errorf("confirmar transação SQLite: %w: %v", ErrStorage, err)
	}
	return nil
}

type accountTransaction struct {
	connection *sql.Conn
}

func (transaction *accountTransaction) FindByID(ctx context.Context, id string) (domain.Account, error) {
	return findByID(ctx, transaction.connection, id)
}

func (transaction *accountTransaction) Update(ctx context.Context, account domain.Account) error {
	return update(ctx, transaction.connection, account)
}

type queryExecutor interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func findByID(ctx context.Context, executor queryExecutor, id string) (domain.Account, error) {
	var account domain.Account
	var status string
	var createdAt string
	var updatedAt string
	err := executor.QueryRowContext(ctx, `
SELECT id, name, document, balance, status, created_at, updated_at
FROM accounts
WHERE id = ?`, id).Scan(
		&account.ID,
		&account.Name,
		&account.Document,
		&account.Balance,
		&status,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, ErrAccountNotFound
	}
	if err != nil {
		return domain.Account{}, fmt.Errorf("buscar conta: %w: %v", ErrStorage, err)
	}
	if !domain.IsValidAccountID(account.ID) || !domain.Status(status).IsValid() {
		return domain.Account{}, fmt.Errorf("ler conta: %w: estado persistido inválido", ErrStorage)
	}
	account.Status = domain.Status(status)
	account.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return domain.Account{}, fmt.Errorf("ler conta: %w: created_at inválido", ErrStorage)
	}
	account.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return domain.Account{}, fmt.Errorf("ler conta: %w: updated_at inválido", ErrStorage)
	}
	return account, nil
}

func update(ctx context.Context, executor execExecutor, account domain.Account) error {
	result, err := executor.ExecContext(ctx, `
UPDATE accounts
SET name = ?, balance = ?, status = ?, updated_at = ?
WHERE id = ?`,
		account.Name,
		account.Balance,
		account.Status,
		formatTime(account.UpdatedAt),
		account.ID,
	)
	if err != nil {
		return fmt.Errorf("atualizar conta: %w: %v", ErrStorage, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar atualização da conta: %w: %v", ErrStorage, err)
	}
	if affected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func rollback(ctx context.Context, connection *sql.Conn) error {
	_, err := connection.ExecContext(ctx, "ROLLBACK")
	return err
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func isDuplicateDocumentError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed: accounts.document")
}
