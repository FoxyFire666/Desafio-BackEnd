package service

import (
	"context"
	"testing"

	"desafio-backend/internal/domain"
	"desafio-backend/internal/repository"
)

func TestWalletLifecycle(t *testing.T) {
	store, err := repository.NewStore("memory")
	if err != nil {
		t.Fatalf("store init failed: %v", err)
	}
	svc := NewService(store)
	wallet, err := svc.OpenWallet(context.Background(), OpenWalletRequest{PlayerID: "player-1", Balance: mustMoney("100.00")})
	if err != nil {
		t.Fatalf("wallet open failed: %v", err)
	}
	if wallet.Balance.String() != "100.00" {
		t.Fatalf("balance mismatch: %s", wallet.Balance.String())
	}
	result, err := svc.CreateWagerTransaction(context.Background(), WagerRequest{
		ProviderID:            "provider-a",
		ExternalTransactionID: "txn-1",
		IdempotencyKey:        "provider-a:txn-1",
		PlayerID:              "player-1",
		WalletID:              wallet.ID,
		RoundID:               "round-1",
		GameID:                "fortune",
		Kind:                  domain.TransactionKindBet,
		Money:                 mustMoney("80.00"),
	})
	if err != nil {
		t.Fatalf("bet failed: %v", err)
	}
	if result.Status != domain.TransactionStatusProcessed {
		t.Fatalf("bet expected PROCESSED, got %s", result.Status)
	}
	stored, err := svc.GetWallet(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("get wallet failed: %v", err)
	}
	if stored.Balance.String() != "20.00" {
		t.Fatalf("final balance mismatch: %s", stored.Balance.String())
	}

	result, err = svc.CreateWagerTransaction(context.Background(), WagerRequest{
		ProviderID:            "provider-a",
		ExternalTransactionID: "txn-2",
		IdempotencyKey:        "provider-a:txn-2",
		PlayerID:              "player-1",
		WalletID:              wallet.ID,
		RoundID:               "round-2",
		GameID:                "fortune",
		Kind:                  domain.TransactionKindBet,
		Money:                 mustMoney("80.00"),
	})
	if err != nil {
		t.Fatalf("rejected bet should not fail service request: %v", err)
	}
	if result.Status != domain.TransactionStatusRejected {
		t.Fatalf("second bet should be REJECTED, got %s", result.Status)
	}
}

func mustMoney(raw string) domain.Money {
	m, err := domain.ParseMoneyString(raw)
	if err != nil {
		panic(err)
	}
	return m
}
