package domain

import "testing"

func TestParseMoneyString(t *testing.T) {
	money, err := ParseMoneyString("25.00")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if money.Amount != 2500 {
		t.Fatalf("expected 2500 minor units, got %d", money.Amount)
	}
	if money.String() != "25.00" {
		t.Fatalf("unexpected string form: %s", money.String())
	}
	if _, err := ParseMoneyString("NaN"); err == nil {
		t.Fatal("NaN should be rejected")
	}
}

func TestWalletOperations(t *testing.T) {
	money, _ := ParseMoneyString("100.00")
	wallet, err := NewWallet("wallet-1", "player-1", money)
	if err != nil {
		t.Fatalf("new wallet failed: %v", err)
	}
	bet, _ := ParseMoneyString("80.00")
	if err := wallet.Debit(bet); err != nil {
		t.Fatalf("debit should succeed: %v", err)
	}
	if wallet.Balance.String() != "20.00" {
		t.Fatalf("final balance mismatch: %s", wallet.Balance.String())
	}
	if wallet.Version != 2 {
		t.Fatalf("version should increment to 2, got %d", wallet.Version)
	}
	if err := wallet.Debit(bet); err == nil {
		t.Fatal("second debit should fail with insufficient funds")
	}
}
