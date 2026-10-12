package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"desafio-backend/internal/domain"
	"desafio-backend/internal/repository"
)

type Service struct {
	Store repository.WalletStore
}

type OpenWalletRequest struct {
	PlayerID string
	Balance  domain.Money
}

type WagerRequest struct {
	ProviderID            string
	ExternalTransactionID string
	IdempotencyKey        string
	PlayerID              string
	WalletID              string
	RoundID               string
	GameID                string
	Kind                  domain.TransactionKind
	Money                 domain.Money
	ReferenceExternalTxID string
}

type WagerResult struct {
	TransactionID    string
	Status           domain.TransactionStatus
	Balance          domain.Money
	IdempotentReplay bool
}

func NewService(store repository.WalletStore) *Service {
	return &Service{Store: store}
}

func (s *Service) OpenWallet(ctx context.Context, req OpenWalletRequest) (domain.Wallet, error) {
	if req.PlayerID == "" {
		return domain.Wallet{}, errors.New("player id is required")
	}
	if req.Balance.Currency == "" {
		req.Balance.Currency = "BRL"
	}
	if req.Balance.Amount < 0 {
		return domain.Wallet{}, errors.New("initial balance cannot be negative")
	}
	_, exists, err := s.Store.GetWalletByPlayerAndCurrency(ctx, req.PlayerID, req.Balance.Currency)
	if err != nil {
		return domain.Wallet{}, err
	}
	if exists {
		return domain.Wallet{}, errors.New("wallet already exists for player and currency")
	}
	walletID := newID("wallet")
	wallet, err := domain.NewWallet(walletID, req.PlayerID, req.Balance)
	if err != nil {
		return domain.Wallet{}, err
	}
	if err := s.Store.CreateWallet(ctx, wallet); err != nil {
		return domain.Wallet{}, err
	}
	if req.Balance.Amount > 0 {
		ledgerEntry := repository.LedgerEntry{
			ID:        newID("ledger"),
			WalletID:  wallet.ID,
			Type:      "OPENING",
			Amount:    req.Balance,
			Balance:   wallet.Balance,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.Store.AppendLedgerEntry(ctx, ledgerEntry); err != nil {
			return domain.Wallet{}, err
		}
		event := repository.OutboxEvent{ID: newID("outbox"), Type: "WalletBalanceChanged", Aggregate: wallet.ID, Payload: fmt.Sprintf(`{"walletId":"%s","transactionId":"%s","direction":"credit","money":"%s","balanceBefore":"0.00","balanceAfter":"%s","walletVersion":1}`, wallet.ID, "OPENING", req.Balance.String(), wallet.Balance.String()), CreatedAt: time.Now().UTC()}
		if err := s.Store.AppendOutbox(ctx, event); err != nil {
			return domain.Wallet{}, err
		}
	}
	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID string) (domain.Wallet, error) {
	w, ok, err := s.Store.GetWalletByID(ctx, walletID)
	if err != nil {
		return domain.Wallet{}, err
	}
	if !ok {
		return domain.Wallet{}, errors.New("wallet not found")
	}
	return w, nil
}

func (s *Service) GetLedger(ctx context.Context, walletID string, limit int, cursor string) ([]repository.LedgerEntry, string, error) {
	return s.Store.ListLedger(ctx, walletID, limit, cursor)
}

func (s *Service) CreateWagerTransaction(ctx context.Context, req WagerRequest) (WagerResult, error) {
	if req.ProviderID == "" || req.ExternalTransactionID == "" {
		return WagerResult{}, errors.New("provider and external transaction id are required")
	}
	if req.IdempotencyKey == "" {
		return WagerResult{}, errors.New("idempotency key is required")
	}
	if req.ProviderID == "" || req.WalletID == "" || req.PlayerID == "" {
		return WagerResult{}, errors.New("wallet and player are required")
	}
	if req.Kind == "" {
		return WagerResult{}, errors.New("kind is required")
	}
	if req.Money.Currency == "" {
		req.Money.Currency = "BRL"
	}
	if req.Money.Amount <= 0 {
		return WagerResult{}, errors.New("money amount must be greater than zero")
	}
	if existing, ok, err := s.Store.GetWagerTransactionByExternalID(ctx, req.ProviderID, req.ExternalTransactionID); err == nil && ok {
		return WagerResult{TransactionID: existing.ID, Status: existing.Status, Balance: existing.ResultMoney, IdempotentReplay: true}, nil
	}
	if existing, ok, err := s.Store.GetWagerTransactionByIdempotencyKey(ctx, req.IdempotencyKey); err == nil && ok {
		if existing.ProviderID == req.ProviderID && existing.ExternalTransactionID == req.ExternalTransactionID {
			return WagerResult{TransactionID: existing.ID, Status: existing.Status, Balance: existing.ResultMoney, IdempotentReplay: true}, nil
		}
		return WagerResult{}, errors.New("idempotency conflict")
	}
	wallet, ok, err := s.Store.GetWalletByID(ctx, req.WalletID)
	if err != nil {
		return WagerResult{}, err
	}
	if !ok {
		return WagerResult{}, errors.New("wallet not found")
	}
	if wallet.PlayerID != req.PlayerID {
		return WagerResult{}, errors.New("wallet does not belong to player")
	}
	transactionID := newID("txn")
	tx := domain.NewWagerTransaction(transactionID, req.ProviderID, req.ExternalTransactionID, req.IdempotencyKey, req.PlayerID, req.WalletID, req.RoundID, req.GameID, req.Kind, req.Money)
	tx.BusinessHash = tx.BusinessKeyHash()
	if err := s.Store.CreateWagerTransaction(ctx, tx); err != nil {
		return WagerResult{}, err
	}
	resultMoney := domain.NewZeroMoney(req.Money.Currency)
	status := domain.TransactionStatusProcessed
	code := ""
	if req.Kind == domain.TransactionKindBet || req.Kind == domain.TransactionKindLoss {
		if !wallet.Balance.GreaterThanOrEqual(req.Money) {
			status = domain.TransactionStatusRejected
			code = "INSUFFICIENT_FUNDS"
		} else {
			if err := wallet.Debit(req.Money); err != nil {
				status = domain.TransactionStatusRejected
				code = "INSUFFICIENT_FUNDS"
			} else {
				resultMoney = req.Money
			}
		}
	} else {
		if err := wallet.Credit(req.Money); err != nil {
			status = domain.TransactionStatusRejected
			code = "INVALID_OPERATION"
		} else {
			resultMoney = req.Money
		}
	}
	if status == domain.TransactionStatusRejected {
		tx.MarkRejected(code)
		if err := s.Store.UpdateWagerTransaction(ctx, tx); err != nil {
			return WagerResult{}, err
		}
		return WagerResult{TransactionID: tx.ID, Status: tx.Status, Balance: domain.NewZeroMoney(req.Money.Currency), IdempotentReplay: false}, nil
	}
	if err := s.Store.SaveWallet(ctx, wallet); err != nil {
		return WagerResult{}, err
	}
	tx.MarkProcessed(resultMoney)
	if err := s.Store.UpdateWagerTransaction(ctx, tx); err != nil {
		return WagerResult{}, err
	}
	ledgerEntry := repository.LedgerEntry{
		ID:        newID("ledger"),
		WalletID:  wallet.ID,
		Type:      string(req.Kind),
		Amount:    req.Money,
		Balance:   wallet.Balance,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Store.AppendLedgerEntry(ctx, ledgerEntry); err != nil {
		return WagerResult{}, err
	}
	event := repository.OutboxEvent{ID: newID("outbox"), Type: "WagerTransactionProcessed", Aggregate: tx.ID, Payload: fmt.Sprintf(`{"transactionId":"%s","walletId":"%s","status":"PROCESSED","amount":"%s"}`, tx.ID, wallet.ID, req.Money.String()), CreatedAt: time.Now().UTC()}
	if err := s.Store.AppendOutbox(ctx, event); err != nil {
		return WagerResult{}, err
	}
	balanceEvent := repository.OutboxEvent{ID: newID("outbox"), Type: "WalletBalanceChanged", Aggregate: wallet.ID, Payload: fmt.Sprintf(`{"walletId":"%s","transactionId":"%s","direction":"%s","money":"%s","balanceBefore":"0.00","balanceAfter":"%s","walletVersion":%d}`, wallet.ID, tx.ID, "debit", req.Money.String(), wallet.Balance.String(), wallet.Version), CreatedAt: time.Now().UTC()}
	if err := s.Store.AppendOutbox(ctx, balanceEvent); err != nil {
		return WagerResult{}, err
	}
	return WagerResult{TransactionID: tx.ID, Status: tx.Status, Balance: wallet.Balance, IdempotentReplay: false}, nil
}

func (s *Service) GetTransactionByID(ctx context.Context, id string) (domain.WagerTransaction, error) {
	tx, ok, err := s.Store.GetWagerTransactionByID(ctx, id)
	if err != nil {
		return domain.WagerTransaction{}, err
	}
	if !ok {
		return domain.WagerTransaction{}, errors.New("transaction not found")
	}
	return tx, nil
}

func (s *Service) GetTransactionByExternalID(ctx context.Context, providerID, externalID string) (domain.WagerTransaction, error) {
	tx, ok, err := s.Store.GetWagerTransactionByExternalID(ctx, providerID, externalID)
	if err != nil {
		return domain.WagerTransaction{}, err
	}
	if !ok {
		return domain.WagerTransaction{}, errors.New("transaction not found")
	}
	return tx, nil
}

func newID(prefix string) string {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buf))
}
