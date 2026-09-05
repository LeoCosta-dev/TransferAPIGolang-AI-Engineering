package domain

import (
	"errors"
	"testing"
)

func TestMoneyRules(t *testing.T) {
	a := Account{Status: StatusActive}
	if err := a.Credit(100); err != nil || a.Balance != 100 {
		t.Fatalf("%d %v", a.Balance, err)
	}
	if err := a.Debit(101); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatal(err)
	}
	if err := a.Debit(100); err != nil || a.Balance != 0 {
		t.Fatalf("%d %v", a.Balance, err)
	}
	for _, status := range []Status{StatusBlocked, StatusClosed} {
		if err := (&Account{Status: status}).Credit(1); err == nil {
			t.Fatalf("status %s aceito", status)
		}
	}
}
