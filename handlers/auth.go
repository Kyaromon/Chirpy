package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Kyaromon/Chirpy/internal/auth"
	"github.com/Kyaromon/Chirpy/internal/database"
	"github.com/google/uuid"
)

type AuthConfig struct {
	DB        *database.Queries
	JWTSecret string
}

func (cfg *AuthConfig) HandlerLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req userRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Email and password required")
		return
	}

	dbUser, err := cfg.DB.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	passwordMatch, err := auth.CheckPasswordHash(req.Password, dbUser.HashedPassword)
	if err != nil || !passwordMatch {
		RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.JWTSecret, 1*time.Hour)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create access token")
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	_, err = cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		ID:        uuid.New(),
		Token:     refreshToken,
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().UTC().Add(60 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		RevokedAt: sql.NullTime{},
		
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to store refresh token")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":            dbUser.ID.String(),
		"email":         dbUser.Email,
		"token":         accessToken,
		"refresh_token": refreshToken,
		"created_at":    dbUser.CreatedAt,
		"updated_at":    dbUser.UpdatedAt,
		"is_chirpy_red": dbUser.IsChirpyRed,
	})
}

func (cfg *AuthConfig) HandlerRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid refresh token")
		return
	}

	dbRefreshToken, err := cfg.DB.GetRefreshToken(r.Context(), refreshToken)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	if dbRefreshToken.RevokedAt.Valid {
		RespondWithError(w, http.StatusUnauthorized, "Refresh token has been revoked")
		return
	}

	if time.Now().UTC().After(dbRefreshToken.ExpiresAt) {
		RespondWithError(w, http.StatusUnauthorized, "Refresh token has expired")
		return
	}

	newAccessToken, err := auth.MakeJWT(dbRefreshToken.UserID, cfg.JWTSecret, 1*time.Hour)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create access token")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token": newAccessToken,
	})
}

func (cfg *AuthConfig) HandlerRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid refresh token")
		return
	}

	now := time.Now().UTC()
	err = cfg.DB.RevokeRefreshToken(r.Context(), database.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{
			Time:  now,
			Valid: true,
		},
		UpdatedAt: now,
		Token:     refreshToken,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to revoke token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}