package domain

import (
	"errors"
	"testing"
	"time"
)

const validAccountID = "550e8400-e29b-41d4-a716-446655440000"

func TestAccountUpdateNameRules(t *testing.T) {
	updatedAt := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		input    string
		wantErr  error
		wantName string
	}{
		{"nome válido é normalizado", "  Novo Nome  ", nil, "Novo Nome"},
		{"nome vazio é rejeitado", "", ErrInvalidName, "Original"},
		{"nome só de espaços é rejeitado", "   ", ErrInvalidName, "Original"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account, err := NewAccount("Original", "123")
			if err != nil {
				t.Fatal(err)
			}

			err = account.UpdateName(tc.input, updatedAt)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erro = %v, want %v", err, tc.wantErr)
			}
			if account.Name != tc.wantName {
				t.Fatalf("name = %q, want %q", account.Name, tc.wantName)
			}
			if tc.wantErr == nil && !account.UpdatedAt.Equal(updatedAt) {
				t.Fatalf("updated_at = %v, want %v", account.UpdatedAt, updatedAt)
			}
		})
	}
}

func TestAccountChangeStatusRules(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		from    Status
		to      Status
		wantErr error
	}{
		{"ativa para bloqueada", StatusActive, StatusBlocked, nil},
		{"ativa para fechada", StatusActive, StatusClosed, nil},
		{"bloqueada para ativa", StatusBlocked, StatusActive, nil},
		{"bloqueada para fechada", StatusBlocked, StatusClosed, nil},
		{"ativa para ativa", StatusActive, StatusActive, ErrInvalidStatusChange},
		{"bloqueada para bloqueada", StatusBlocked, StatusBlocked, ErrInvalidStatusChange},
		{"fechada para ativa", StatusClosed, StatusActive, ErrInvalidStatusChange},
		{"fechada para bloqueada", StatusClosed, StatusBlocked, ErrInvalidStatusChange},
		{"fechada para fechada", StatusClosed, StatusClosed, ErrInvalidStatusChange},
		{"status inexistente", StatusActive, Status("PAUSED"), ErrInvalidStatus},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := Account{Status: tc.from}

			err := account.ChangeStatus(tc.to, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("erro = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if account.Status != tc.to {
					t.Fatalf("status = %q, want %q", account.Status, tc.to)
				}
				if !account.UpdatedAt.Equal(now.UTC()) {
					t.Fatalf("updated_at = %v, want %v", account.UpdatedAt, now.UTC())
				}
			}
		})
	}
}

func TestIsValidAccountID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"uuid v4 válido", validAccountID, true},
		{"variante maiúscula", "550e8400-e29b-41d4-A716-446655440000", true},
		{"vazio", "", false},
		{"comprimento menor", "550e8400-e29b-41d4-a716-44665544000", false},
		{"comprimento maior", "550e8400-e29b-41d4-a716-4466554400000", false},
		{"tracejado ausente", "550e8400Xe29b-41d4-a716-446655440000", false},
		{"caractere não hexadecimal", "550e8400-e29b-g1d4-a716-446655440000", false},
		{"versão inválida", "550e8400-e29b-51d4-a716-446655440000", false},
		{"variante inválida", "550e8400-e29b-41d4-c716-446655440000", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidAccountID(tc.id); got != tc.want {
				t.Fatalf("IsValidAccountID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestValidateAccountIDContract(t *testing.T) {
	if err := ValidateAccountID(validAccountID); err != nil {
		t.Fatalf("erro = %v", err)
	}
	if err := ValidateAccountID("nao-e-uuid"); !errors.Is(err, ErrInvalidAccountID) {
		t.Fatalf("erro = %v", err)
	}
}

func TestNewAccountIDGeneratesUniqueValidIdentifiers(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		id, err := NewAccountID()
		if err != nil {
			t.Fatal(err)
		}
		if !IsValidAccountID(id) {
			t.Fatalf("identificador inválido gerado: %q", id)
		}
		if seen[id] {
			t.Fatalf("identificador duplicado gerado: %q", id)
		}
		seen[id] = true
	}
}

func TestNewAccountAtNormalizesAndUsesUTC(t *testing.T) {
	location := time.FixedZone("BRT", -3*60*60)
	now := time.Date(2026, 9, 5, 9, 30, 0, 0, location)

	account, err := NewAccountAt(" Nome ", " Doc ", now)
	if err != nil {
		t.Fatal(err)
	}

	if account.Name != "Nome" || account.Document != "Doc" {
		t.Fatalf("dados = %q/%q, want Nome/Doc", account.Name, account.Document)
	}
	if account.Status != StatusActive {
		t.Fatalf("status = %q, want %q", account.Status, StatusActive)
	}
	if !account.CreatedAt.Equal(now.UTC()) || account.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at = %v (%v), want %v em UTC", account.CreatedAt, account.CreatedAt.Location(), now.UTC())
	}
	if !account.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("updated_at = %v, want %v", account.UpdatedAt, now.UTC())
	}
	if !IsValidAccountID(account.ID) {
		t.Fatalf("identificador inválido: %q", account.ID)
	}
}
