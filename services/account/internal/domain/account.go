package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

const uuidV4Length = 36

type Account struct {
	ID        string
	Name      string
	Document  string
	Balance   int64
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewAccount(name, document string) (Account, error) {
	return NewAccountAt(name, document, time.Now().UTC())
}

func NewAccountAt(name, document string, now time.Time) (Account, error) {
	name, document, err := validateAccountData(name, document)
	if err != nil {
		return Account{}, err
	}

	id, err := NewAccountID()
	if err != nil {
		return Account{}, fmt.Errorf("gerar identificador da conta: %w", err)
	}

	now = now.UTC()
	return Account{
		ID:        id,
		Name:      name,
		Document:  document,
		Balance:   0,
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NewAccountID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16])), nil
}

func IsValidAccountID(id string) bool {
	if len(id) != uuidV4Length || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}

	for index := 0; index < len(id); index++ {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		if !isHexDigit(id[index]) {
			return false
		}
	}

	return id[14] == '4' && (id[19] == '8' || id[19] == '9' || id[19] == 'a' || id[19] == 'b' || id[19] == 'A' || id[19] == 'B')
}

func ValidateAccountID(id string) error {
	if !IsValidAccountID(id) {
		return ErrInvalidAccountID
	}
	return nil
}

func (account *Account) UpdateName(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}

	account.Name = name
	account.UpdatedAt = now.UTC()
	return nil
}

func (account *Account) ChangeStatus(status Status, now time.Time) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}
	if !canTransition(account.Status, status) {
		return ErrInvalidStatusChange
	}
	if status == StatusClosed && account.Balance != 0 {
		return ErrInvalidStatusChange
	}

	account.Status = status
	account.UpdatedAt = now.UTC()
	return nil
}

func (account *Account) Credit(amount int64, now time.Time) error {
	if amount <= 0 || account.Balance > math.MaxInt64-amount {
		return ErrInvalidAmount
	}
	if account.Status == StatusBlocked {
		return ErrAccountBlocked
	}
	if account.Status == StatusClosed {
		return ErrAccountClosed
	}

	account.Balance += amount
	account.UpdatedAt = now.UTC()
	return nil
}

func (account *Account) Debit(amount int64, now time.Time) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if account.Status == StatusBlocked {
		return ErrAccountBlocked
	}
	if account.Status == StatusClosed {
		return ErrAccountClosed
	}
	if amount > account.Balance {
		return ErrInsufficientBalance
	}

	account.Balance -= amount
	account.UpdatedAt = now.UTC()
	return nil
}

func validateAccountData(name, document string) (string, string, error) {
	name = strings.TrimSpace(name)
	document = strings.TrimSpace(document)
	if name == "" {
		return "", "", ErrInvalidName
	}
	if document == "" {
		return "", "", ErrInvalidDocument
	}
	return name, document, nil
}

func isHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f') || (value >= 'A' && value <= 'F')
}
