package domain

import (
	"errors"
	"math"
	"testing"
)

func TestCreditRules(t *testing.T) {
	cases := []struct {
		name        string
		account     Account
		amount      int64
		wantErr     error
		wantBalance int64
	}{
		{"crédito válido", Account{Status: StatusActive}, 100, nil, 100},
		{"crédito sobre saldo existente", Account{Status: StatusActive, Balance: 50}, 25, nil, 75},
		{"valor zero", Account{Status: StatusActive}, 0, ErrInvalidAmount, 0},
		{"valor negativo", Account{Status: StatusActive}, -10, ErrInvalidAmount, 0},
		{"estouro de inteiro", Account{Status: StatusActive, Balance: math.MaxInt64 - 5}, 10, ErrInvalidAmount, math.MaxInt64 - 5},
		{"conta bloqueada", Account{Status: StatusBlocked}, 1, ErrAccountBlocked, 0},
		{"conta fechada", Account{Status: StatusClosed}, 1, ErrAccountClosed, 0},
		{"status desconhecido não movimenta", Account{Status: Status("PAUSED")}, 1, ErrAccountBlocked, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.account.Credit(tc.amount)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erro = %v, want %v", err, tc.wantErr)
			}
			if tc.account.Balance != tc.wantBalance {
				t.Fatalf("balance = %d, want %d", tc.account.Balance, tc.wantBalance)
			}
		})
	}
}

func TestDebitRules(t *testing.T) {
	cases := []struct {
		name        string
		account     Account
		amount      int64
		wantErr     error
		wantBalance int64
	}{
		{"débito válido", Account{Status: StatusActive, Balance: 100}, 30, nil, 70},
		{"débito do saldo total", Account{Status: StatusActive, Balance: 100}, 100, nil, 0},
		{"saldo insuficiente", Account{Status: StatusActive, Balance: 100}, 101, ErrInsufficientBalance, 100},
		{"valor zero", Account{Status: StatusActive, Balance: 100}, 0, ErrInvalidAmount, 100},
		{"valor negativo", Account{Status: StatusActive, Balance: 100}, -1, ErrInvalidAmount, 100},
		{"conta bloqueada", Account{Status: StatusBlocked, Balance: 100}, 1, ErrAccountBlocked, 100},
		{"conta fechada", Account{Status: StatusClosed, Balance: 100}, 1, ErrAccountClosed, 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.account.Debit(tc.amount)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erro = %v, want %v", err, tc.wantErr)
			}
			if tc.account.Balance != tc.wantBalance {
				t.Fatalf("balance = %d, want %d", tc.account.Balance, tc.wantBalance)
			}
		})
	}
}

func TestStatusValidation(t *testing.T) {
	cases := []struct {
		status Status
		want   bool
	}{
		{StatusActive, true},
		{StatusBlocked, true},
		{StatusClosed, true},
		{Status(""), false},
		{Status("PAUSED"), false},
		{Status("active"), false},
	}

	for _, tc := range cases {
		if got := tc.status.IsValid(); got != tc.want {
			t.Fatalf("IsValid(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusActive, StatusBlocked, true},
		{StatusActive, StatusClosed, true},
		{StatusBlocked, StatusActive, true},
		{StatusBlocked, StatusClosed, true},
		{StatusActive, StatusActive, false},
		{StatusBlocked, StatusBlocked, false},
		{StatusClosed, StatusActive, false},
		{StatusClosed, StatusBlocked, false},
		{StatusClosed, StatusClosed, false},
		{Status("PAUSED"), StatusActive, false},
		{StatusActive, Status("PAUSED"), false},
	}

	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
