package transaction

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/morazss/fintracker/internal/account"
	"github.com/morazss/fintracker/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, account.ErrNotFound):
			http.Error(w, "account not found", http.StatusNotFound)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}
