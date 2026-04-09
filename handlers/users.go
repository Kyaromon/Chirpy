package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Kyaromon/Chirpy/internal/auth"
	"github.com/Kyaromon/Chirpy/internal/database"
)

type User struct {
	ID        		string    `json:"id"`
	Email     		string    `json:"email"`
	IsChirpyRed     bool      `json:"is_chirpy_red"`
	CreatedAt 		time.Time `json:"created_at"`
	UpdatedAt 		time.Time `json:"updated_at"`
}

type userRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserConfig struct {
	DB        *database.Queries
	JWTSecret string
}

func (cfg *UserConfig) HandlerUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		cfg.HandlerCreateUser(w, r)
		return
	}
	if r.Method == http.MethodPut {
		cfg.HandlerUpdateUser(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (cfg *UserConfig) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {
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

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	dbUser, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	user := User{
		ID:           dbUser.ID.String(),
		Email:        dbUser.Email,
		IsChirpyRed:  dbUser.IsChirpyRed,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
	}

	RespondWithJSON(w, http.StatusCreated, user)
}

func (cfg *UserConfig) HandlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var req userRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&req)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Email and password required")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	dbUser, err := cfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPassword,
		UpdatedAt:      time.Now().UTC(),
		ID:             userID,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	user := User{
		ID:           dbUser.ID.String(),
		Email:        dbUser.Email,
		IsChirpyRed:  dbUser.IsChirpyRed,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
	}

	RespondWithJSON(w, http.StatusOK, user)
}