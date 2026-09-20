package transactions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ilijavlahovic24-bit/azurebank/internal/auth"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

type amountRequest struct {
	AccountID int64 `json:"account_id"`
	Amount    int64 `json:"amount"`
}

type transferRequest struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.AccountID <= 0 || req.Amount <= 0 {
		http.Error(w, "account_id and amount must be positive", http.StatusBadRequest)
		return
	}

	t, err := h.store.Deposit(r.Context(), userID, req.AccountID, req.Amount)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.AccountID <= 0 || req.Amount <= 0 {
		http.Error(w, "account_id and amount must be positive", http.StatusBadRequest)
		return
	}

	t, err := h.store.Withdraw(r.Context(), userID, req.AccountID, req.Amount)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.FromAccountID <= 0 || req.ToAccountID <= 0 || req.Amount <= 0 {
		http.Error(w, "from_account_id, to_account_id and amount must be positive", http.StatusBadRequest)
		return
	}

	t, err := h.store.Transfer(r.Context(), userID, req.FromAccountID, req.ToAccountID, req.Amount)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || accountID <= 0 {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	txs, err := h.store.ListByAccount(r.Context(), userID, accountID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		http.Error(w, "account not found", http.StatusNotFound)
	case errors.Is(err, ErrInsufficientFunds):
		http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
	case errors.Is(err, ErrSameAccount):
		http.Error(w, "from and to accounts must be different", http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
