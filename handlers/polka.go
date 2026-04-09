package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Kyaromon/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/Kyaromon/Chirpy/internal/auth"
)

type PolkaConfig struct {
	DB       *database.Queries
	APIKey    string
}

type polkaWebhookRequest struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *PolkaConfig) HandlerPolkaWebhooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if apiKey != cfg.APIKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req polkaWebhookRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(req.Data.UserID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = cfg.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = cfg.DB.UpgradeUserToChirpyRed(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}