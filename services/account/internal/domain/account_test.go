package domain

import (
	"errors"
	"testing"
	"time"
)

func TestAccountLifecycle(t *testing.T) {
	account, err := NewAccountAt(" Nome ", " Documento ", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if account.Name != "Nome" || account.Document != "Documento" || account.Status != StatusActive || !IsValidAccountID(account.ID) {
		t.Fatalf("conta inválida: %+v", account)
	}
	if err := account.ChangeStatus(StatusBlocked, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := account.ChangeStatus(StatusActive, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := account.ChangeStatus(StatusClosed, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := account.ChangeStatus(StatusActive, time.Now()); !errors.Is(err, ErrInvalidStatusChange) {
		t.Fatalf("erro = %v", err)
	}
}

func TestAccountRejectsInvalidData(t *testing.T) {
	if _, err := NewAccount(" ", "x"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("erro = %v", err)
	}
	if _, err := NewAccount("x", " "); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("erro = %v", err)
	}
}
