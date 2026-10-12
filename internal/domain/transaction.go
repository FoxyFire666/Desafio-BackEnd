package domain

import (
	"fmt"
	"time"
)

type TransactionKind string

const (
	TransactionKindOpening  TransactionKind = "OPENING"
	TransactionKindBet      TransactionKind = "BET"
	TransactionKindWin      TransactionKind = "WIN"
	TransactionKindLoss     TransactionKind = "LOSS"
	TransactionKindRefund   TransactionKind = "REFUND"
	TransactionKindRollback TransactionKind = "ROLLBACK"
)

type TransactionStatus string

const (
	TransactionStatusPending             TransactionStatus = "PENDING"
	TransactionStatusProcessed           TransactionStatus = "PROCESSED"
	TransactionStatusRejected            TransactionStatus = "REJECTED"
	TransactionStatusFailed              TransactionStatus = "FAILED"
	TransactionStatusPendingReference    TransactionStatus = "PENDING_REFERENCE"
)

type WagerTransaction struct {
	ID                     string            `json:"id"`
	ProviderID             string            `json:"providerId"`
	ExternalTransactionID  string            `json:"externalTransactionId"`
	IdempotencyKey         string            `json:"idempotencyKey"`
	PlayerID               string            `json:"playerId"`
	WalletID               string            `json:"walletId"`
	RoundID                string            `json:"roundId"`
	GameID                 string            `json:"gameId"`
	Kind                   TransactionKind   `json:"kind"`
	Money                 Money             `json:"money"`
	ReferenceExternalTxID  string            `json:"referenceExternalTransactionId,omitempty"`
	Status                 TransactionStatus `json:"status"`
	FailureCode            string            `json:"failureCode,omitempty"`
	ResultMoney            Money             `json:"resultMoney,omitempty"`
	CreatedAt              time.Time         `json:"createdAt"`
	UpdatedAt              time.Time         `json:"updatedAt"`
	ResolvedInternalTxID   string            `json:"resolvedInternalTransactionId,omitempty"`
	BusinessHash           string            `json:"businessHash,omitempty"`
}

func NewWagerTransaction(id, providerID, externalTransactionID, idempotencyKey, playerID, walletID, roundID, gameID string, kind TransactionKind, money Money) WagerTransaction {
	return WagerTransaction{
		ID:                    id,
		ProviderID:            providerID,
		ExternalTransactionID: externalTransactionID,
		IdempotencyKey:        idempotencyKey,
		PlayerID:              playerID,
		WalletID:              walletID,
		RoundID:               roundID,
		GameID:                gameID,
		Kind:                  kind,
		Money:                 money,
		Status:                TransactionStatusPending,
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}
}

func (t *WagerTransaction) MarkProcessed(result Money) {
	t.Status = TransactionStatusProcessed
	t.ResultMoney = result
	t.UpdatedAt = time.Now().UTC()
}

func (t *WagerTransaction) MarkRejected(code string) {
	t.Status = TransactionStatusRejected
	t.FailureCode = code
	t.UpdatedAt = time.Now().UTC()
}

func (t *WagerTransaction) MarkPendingReference() {
	t.Status = TransactionStatusPendingReference
	t.UpdatedAt = time.Now().UTC()
}

func (t *WagerTransaction) BusinessKeyHash() string {
	if t.ProviderID == "" || t.ExternalTransactionID == "" || t.PlayerID == "" || t.WalletID == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s", t.ProviderID, t.ExternalTransactionID, t.PlayerID, t.WalletID, t.RoundID, t.GameID, t.Kind, t.Money.String())
}

func (t WagerTransaction) IsDebit() bool {
	return t.Kind == TransactionKindBet || t.Kind == TransactionKindLoss
}

func (t WagerTransaction) IsCredit() bool {
	return t.Kind == TransactionKindWin || t.Kind == TransactionKindRefund || t.Kind == TransactionKindRollback || t.Kind == TransactionKindOpening
}
