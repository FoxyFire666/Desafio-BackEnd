package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"desafio-backend/internal/auth"
	"desafio-backend/internal/config"
	"desafio-backend/internal/domain"
	"desafio-backend/internal/service"
)

type Server struct {
	Mux     *http.ServeMux
	Service *service.Service
	Auth    *auth.Authenticator
}

func NewServer(cfg config.Config, svc *service.Service) *Server {
	mux := http.NewServeMux()
	auth := auth.New(cfg)
	s := &Server{Mux: mux, Service: svc, Auth: auth}

	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/wallets", s.withAuth(s.handleWallets))
	mux.HandleFunc("/wallets/", s.withAuth(s.handleWalletDetails))
	mux.HandleFunc("/wagering/transactions", s.withAuth(s.handleWagerTransactions))
	mux.HandleFunc("/providers/", s.withAuth(s.handleProviderTransactions))
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Mux.ServeHTTP(w, r)
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.Auth.Enabled() {
			next(w, r)
			return
		}
		claims, ok := s.Auth.Authenticate(r)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := auth.WithContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *Server) handleWallets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var in struct {
			PlayerID       string        `json:"playerId"`
			InitialBalance domain.Money `json:"initialBalance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		wallet, err := s.Service.OpenWallet(r.Context(), service.OpenWalletRequest{PlayerID: in.PlayerID, Balance: in.InitialBalance})
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": wallet.ID, "playerId": wallet.PlayerID, "balance": wallet.Balance, "version": wallet.Version})
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleWalletDetails(w http.ResponseWriter, r *http.Request) {
	walletID := strings.TrimPrefix(r.URL.Path, "/wallets/")
	if walletID == "" || strings.Contains(walletID, "/") {
		walletID = strings.Split(walletID, "/")[0]
	}
	if r.Method == http.MethodGet {
		wallet, err := s.Service.GetWallet(r.Context(), walletID)
		if err != nil {
			http.Error(w, `{"error":"wallet not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(wallet)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleWagerTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var in struct {
			ProviderID             string                `json:"providerId"`
			ExternalTransactionID  string                `json:"externalTransactionId"`
			PlayerID               string                `json:"playerId"`
			WalletID               string                `json:"walletId"`
			RoundID                string                `json:"roundId"`
			GameID                 string                `json:"gameId"`
			Kind                   domain.TransactionKind `json:"kind"`
			Money                  domain.Money          `json:"money"`
			ReferenceExternalTxID  string                `json:"referenceExternalTransactionId,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, `{"error":"idempotency key required"}`, http.StatusBadRequest)
			return
		}
		result, err := s.Service.CreateWagerTransaction(r.Context(), service.WagerRequest{
			ProviderID:             in.ProviderID,
			ExternalTransactionID: in.ExternalTransactionID,
			IdempotencyKey:        key,
			PlayerID:              in.PlayerID,
			WalletID:              in.WalletID,
			RoundID:               in.RoundID,
			GameID:                in.GameID,
			Kind:                  in.Kind,
			Money:                 in.Money,
			ReferenceExternalTxID: in.ReferenceExternalTxID,
		})
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
			return
		}
		resp := map[string]any{"transactionId": result.TransactionID, "status": result.Status, "balance": result.Balance, "idempotentReplay": result.IdempotentReplay}
		json.NewEncoder(w).Encode(resp)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleProviderTransactions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/providers/"), "/")
	if len(parts) < 2 || parts[1] != "wagering" || parts[2] != "transactions" {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	providerID := parts[0]
	externalID := strings.TrimPrefix(r.URL.Path, "/providers/"+providerID+"/wagering/transactions/")
	if r.Method == http.MethodGet {
		tx, err := s.Service.GetTransactionByExternalID(r.Context(), providerID, externalID)
		if err != nil {
			http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(tx)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}
