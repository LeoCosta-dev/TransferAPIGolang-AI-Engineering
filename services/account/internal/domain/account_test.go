package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewAccountTrimsDataAndGeneratesUUIDV4(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	account, err := NewAccountAt("  Leonardo Costa  ", " 123  ", now)
	if err != nil {
		t.Fatalf("NewAccountAt() error = %v", err)
	}

	if account.Name != "Leonardo Costa" || account.Document != "123" {
		t.Fatalf("dados normalizados incorretamente: %+v", account)
	}
	if !IsValidAccountID(account.ID) || account.ID[14] != '4' {
		t.Fatalf("ID não é um UUID v4 válido: %q", account.ID)
	}
	if account.Status != StatusActive || account.Balance != 0 {
		t.Fatalf("estado inicial incorreto: %+v", account)
	}
	if !account.CreatedAt.Equal(now.UTC()) || !account.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("timestamps incorretos: %+v", account)
	}
}

func TestNewAccountRejectsBlankData(t *testing.T) {
	for _, test := range []struct {
		name     string
		document string
		wantErr  error
	}{
		{name: "   ", document: "123", wantErr: ErrInvalidName},
		{name: "Nome", document: " \t", wantErr: ErrInvalidDocument},
	} {
		_, err := NewAccountAt(test.name, test.document, time.Now())
		if !errors.Is(err, test.wantErr) {
			t.Errorf("erro = %v, want %v", err, test.wantErr)
		}
	}
}

func TestAccountStatusTransitions(t *testing.T) {
	now := time.Now()
	account, err := NewAccountAt("Nome", "123", now)
	if err != nil {
		t.Fatal(err)
	}

	if err := account.ChangeStatus(StatusBlocked, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := account.ChangeStatus(StatusActive, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := account.ChangeStatus(StatusClosed, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	previous := account
	if err := account.ChangeStatus(StatusActive, now.Add(4*time.Minute)); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("erro = %v, want %v", err, ErrInvalidStatusChange)
	}
	if account != previous {
		t.Fatal("transição inválida modificou a conta")
	}
}

func TestAccountCannotCloseWithBalance(t *testing.T) {
	account, err := NewAccountAt("Nome", "123", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := account.Credit(100, time.Now()); err != nil {
		t.Fatal(err)
	}
	previous := account
	if err := account.ChangeStatus(StatusClosed, time.Now()); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("erro = %v, want %v", err, ErrInvalidStatusChange)
	}
	if account != previous {
		t.Fatal("fechamento inválido modificou a conta")
	}
}

func TestCreditAndDebitUpdateBalanceAndTimestamp(t *testing.T) {
	initial := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	creditTime := initial.Add(time.Minute)
	debitTime := initial.Add(2 * time.Minute)
	account, err := NewAccountAt("Nome", "123", initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.Credit(100, creditTime); err != nil {
		t.Fatal(err)
	}
	if account.Balance != 100 || !account.UpdatedAt.Equal(creditTime) {
		t.Fatalf("crédito incorreto: %+v", account)
	}
	if err := account.Debit(40, debitTime); err != nil {
		t.Fatal(err)
	}
	if account.Balance != 60 || !account.UpdatedAt.Equal(debitTime) {
		t.Fatalf("débito incorreto: %+v", account)
	}
}

func TestFailedMoneyOperationsDoNotMutateAccount(t *testing.T) {
	account, err := NewAccountAt("Nome", "123", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	previous := account
	if err := account.Debit(1, time.Now().Add(time.Minute)); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("erro = %v, want %v", err, ErrInsufficientBalance)
	}
	if account != previous {
		t.Fatal("débito recusado modificou a conta")
	}
	if err := account.Credit(0, time.Now().Add(2*time.Minute)); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("erro = %v, want %v", err, ErrInvalidAmount)
	}
	if account != previous {
		t.Fatal("crédito recusado modificou a conta")
	}
}

func TestBlockedAndClosedAccountsCannotMoveMoney(t *testing.T) {
	for _, status := range []Status{StatusBlocked, StatusClosed} {
		account, err := NewAccountAt("Nome", "123", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		account.Status = status
		wantErr := ErrAccountBlocked
		if status == StatusClosed {
			wantErr = ErrAccountClosed
		}
		if err := account.Credit(1, time.Now()); !errors.Is(err, wantErr) {
			t.Errorf("status %s: crédito retornou %v, want %v", status, err, wantErr)
		}
	}
}

func TestIsValidAccountIDRejectsNonUUIDV4(t *testing.T) {
	for _, id := range []string{
		"",
		"550e8400-e29b-11d4-a716-446655440000",
		"550e8400-e29b-41d4-6716-446655440000",
		"550e8400e29b41d4a716446655440000",
	} {
		if IsValidAccountID(id) {
			t.Errorf("ID inválido aceito: %q", id)
		}
	}
}
