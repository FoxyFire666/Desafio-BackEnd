package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"desafio-backend/internal/domain"
)

type WalletStore interface {
	CreateWallet(ctx context.Context, w domain.Wallet) error
	GetWalletByID(ctx context.Context, id string) (domain.Wallet, bool, error)
	GetWalletByPlayerAndCurrency(ctx context.Context, playerID, currency string) (domain.Wallet, bool, error)
	SaveWallet(ctx context.Context, w domain.Wallet) error
	CreateWagerTransaction(ctx context.Context, tx domain.WagerTransaction) error
	UpdateWagerTransaction(ctx context.Context, tx domain.WagerTransaction) error
	GetWagerTransactionByExternalID(ctx context.Context, providerID, externalID string) (domain.WagerTransaction, bool, error)
	GetWagerTransactionByID(ctx context.Context, id string) (domain.WagerTransaction, bool, error)
	GetWagerTransactionByIdempotencyKey(ctx context.Context, key string) (domain.WagerTransaction, bool, error)
	AppendLedgerEntry(ctx context.Context, entry LedgerEntry) error
	ListLedger(ctx context.Context, walletID string, limit int, cursor string) ([]LedgerEntry, string, error)
	AppendOutbox(ctx context.Context, event OutboxEvent) error
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, eventID string) error
}

type LedgerEntry struct {
	ID        string       `json:"id"`
	WalletID  string       `json:"walletId"`
	Type      string       `json:"type"`
	Amount    domain.Money `json:"amount"`
	Balance   domain.Money `json:"balance"`
	CreatedAt time.Time    `json:"createdAt"`
}

type OutboxEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Aggregate string    `json:"aggregate"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
	Published bool      `json:"published"`
}

type Store struct {
	mu                  sync.Mutex
	wallets             map[string]domain.Wallet
	walletByPlayer      map[string]domain.Wallet
	transactionsByID    map[string]domain.WagerTransaction
	transactionsByExtID map[string]domain.WagerTransaction
	transactionsByKey   map[string]domain.WagerTransaction
	ledger              map[string][]LedgerEntry
	outbox              map[string]OutboxEvent
	pendingOutbox       []OutboxEvent
	db                  *sql.DB
}

func NewStore(dsn string) (*Store, error) {
	if dsn == "" || dsn == "memory" {
		return &Store{
			wallets:             make(map[string]domain.Wallet),
			walletByPlayer:      make(map[string]domain.Wallet),
			transactionsByID:    make(map[string]domain.WagerTransaction),
			transactionsByExtID: make(map[string]domain.WagerTransaction),
			transactionsByKey:   make(map[string]domain.WagerTransaction),
			ledger:              make(map[string][]LedgerEntry),
			outbox:              make(map[string]OutboxEvent),
		}, nil
	}
	if dsn == "" {
		return nil, errors.New("database dsn is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, wallets: make(map[string]domain.Wallet), walletByPlayer: make(map[string]domain.Wallet), transactionsByID: make(map[string]domain.WagerTransaction), transactionsByExtID: make(map[string]domain.WagerTransaction), transactionsByKey: make(map[string]domain.WagerTransaction), ledger: make(map[string][]LedgerEntry), outbox: make(map[string]OutboxEvent)}, nil
}

func (s *Store) CreateWallet(ctx context.Context, w domain.Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.wallets[w.ID]; exists {
		return fmt.Errorf("wallet already exists")
	}
	key := w.PlayerID + ":" + w.Currency
	if _, exists := s.walletByPlayer[key]; exists {
		return fmt.Errorf("wallet for player/currency exists")
	}
	s.wallets[w.ID] = w
	s.walletByPlayer[key] = w
	return nil
}

func (s *Store) GetWalletByID(ctx context.Context, id string) (domain.Wallet, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.wallets[id]
	return w, ok, nil
}

func (s *Store) GetWalletByPlayerAndCurrency(ctx context.Context, playerID, currency string) (domain.Wallet, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.walletByPlayer[playerID+":"+currency]
	return w, ok, nil
}

func (s *Store) SaveWallet(ctx context.Context, w domain.Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wallets[w.ID] = w
	key := w.PlayerID + ":" + w.Currency
	s.walletByPlayer[key] = w
	return nil
}

func (s *Store) CreateWagerTransaction(ctx context.Context, tx domain.WagerTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.transactionsByID[tx.ID]; exists {
		return fmt.Errorf("transaction already exists")
	}
	key := tx.IdempotencyKey
	if tx.IdempotencyKey != "" {
		if _, exists := s.transactionsByKey[key]; exists {
			return fmt.Errorf("idempotency key already exists")
		}
		s.transactionsByKey[key] = tx
	}
	if tx.ExternalTransactionID != "" {
		extKey := tx.ProviderID + ":" + tx.ExternalTransactionID
		s.transactionsByExtID[extKey] = tx
	}
	s.transactionsByID[tx.ID] = tx
	return nil
}

func (s *Store) UpdateWagerTransaction(ctx context.Context, tx domain.WagerTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactionsByID[tx.ID] = tx
	if tx.IdempotencyKey != "" {
		s.transactionsByKey[tx.IdempotencyKey] = tx
	}
	if tx.ExternalTransactionID != "" {
		s.transactionsByExtID[tx.ProviderID+":"+tx.ExternalTransactionID] = tx
	}
	return nil
}

func (s *Store) GetWagerTransactionByExternalID(ctx context.Context, providerID, externalID string) (domain.WagerTransaction, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, ok := s.transactionsByExtID[providerID+":"+externalID]
	return tx, ok, nil
}

func (s *Store) GetWagerTransactionByID(ctx context.Context, id string) (domain.WagerTransaction, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, ok := s.transactionsByID[id]
	return tx, ok, nil
}

func (s *Store) GetWagerTransactionByIdempotencyKey(ctx context.Context, key string) (domain.WagerTransaction, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, ok := s.transactionsByKey[key]
	return tx, ok, nil
}

func (s *Store) AppendLedgerEntry(ctx context.Context, entry LedgerEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ledger[entry.WalletID] = append(s.ledger[entry.WalletID], entry)
	return nil
}

func (s *Store) ListLedger(ctx context.Context, walletID string, limit int, cursor string) ([]LedgerEntry, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := append([]LedgerEntry(nil), s.ledger[walletID]...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	if cursor != "" {
		for i, entry := range entries {
			if entry.ID == cursor {
				entries = entries[i+1:]
				break
			}
		}
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(entries) > limit {
		return entries[:limit], entries[limit-1].ID, nil
	}
	return entries, "", nil
}

func (s *Store) AppendOutbox(ctx context.Context, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbox[event.ID] = event
	s.pendingOutbox = append(s.pendingOutbox, event)
	return nil
}

func (s *Store) ListPendingOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	pending := make([]OutboxEvent, 0, len(s.pendingOutbox))
	for _, event := range s.pendingOutbox {
		if !event.Published {
			pending = append(pending, event)
		}
	}
	if len(pending) > limit {
		pending = pending[:limit]
	}
	return pending, nil
}

func (s *Store) MarkOutboxPublished(ctx context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event, ok := s.outbox[eventID]; ok {
		event.Published = true
		s.outbox[eventID] = event
	}
	for i := range s.pendingOutbox {
		if s.pendingOutbox[i].ID == eventID {
			s.pendingOutbox[i].Published = true
			break
		}
	}
	return nil
}
