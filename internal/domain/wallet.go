package domain

import (
	"fmt"
	"strings"
	"time"
)

type Wallet struct {
	ID        string    `json:"id"`
	PlayerID  string    `json:"playerId"`
	Currency  string    `json:"currency"`
	Balance   Money     `json:"balance"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewWallet(id, playerID string, initialBalance Money) (Wallet, error) {
	if id == "" {
		return Wallet{}, fmt.Errorf("wallet id is required")
	}
	if playerID == "" {
		return Wallet{}, fmt.Errorf("player id is required")
	}
	if initialBalance.Currency == "" {
		initialBalance.Currency = defaultCurrency
	}
	if initialBalance.Amount < 0 {
		return Wallet{}, fmt.Errorf("initial balance cannot be negative")
	}
	now := time.Now().UTC()
	return Wallet{
		ID:        id,
		PlayerID:  playerID,
		Currency:  strings.ToUpper(initialBalance.Currency),
		Balance:   initialBalance,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (w *Wallet) Credit(amount Money) error {
	if !w.Balance.SameCurrency(amount) {
		return fmt.Errorf("currency mismatch")
	}
	if amount.Amount < 0 {
		return fmt.Errorf("credit amount must be positive")
	}
	newBalance, err := w.Balance.Add(amount)
	if err != nil {
		return err
	}
	w.Balance = newBalance
	w.Version++
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (w *Wallet) Debit(amount Money) error {
	if !w.Balance.SameCurrency(amount) {
		return fmt.Errorf("currency mismatch")
	}
	if amount.Amount < 0 {
		return fmt.Errorf("debit amount must be positive")
	}
	if !w.Balance.GreaterThanOrEqual(amount) {
		return fmt.Errorf("insufficient funds")
	}
	newBalance, err := w.Balance.Subtract(amount)
	if err != nil {
		return err
	}
	if newBalance.Amount < 0 {
		return fmt.Errorf("wallet balance cannot be negative")
	}
	w.Balance = newBalance
	w.Version++
	w.UpdatedAt = time.Now().UTC()
	return nil
}
