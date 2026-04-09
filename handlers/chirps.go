package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/Kyaromon/Chirpy/internal/auth"
	"github.com/Kyaromon/Chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    string    `json:"user_id"`
}

type chirpRequest struct {
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}

type ChirpConfig struct {
	DB        *database.Queries
	JWTSecret string
}

func (cfg *ChirpConfig) HandlerChirps(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")

	if chirpID == "" {
		if r.Method == http.MethodPost {
			cfg.HandlerCreateChirp(w, r)
			return
		}
		if r.Method == http.MethodGet {
			cfg.HandlerGetChirps(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodGet {
		cfg.HandlerGetChirp(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		cfg.HandlerDeleteChirp(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (cfg *ChirpConfig) HandlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	var req chirpRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&req)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Body) > 140 {
		RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	if len(req.Body) == 0 {
		RespondWithError(w, http.StatusBadRequest, "Chirp body cannot be empty")
		return
	}

	cleanedBody := cleanChirp(req.Body)

	dbChirp, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID.String(),
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID.String(),
	}

	RespondWithJSON(w, http.StatusCreated, chirp)
}

func (cfg *ChirpConfig) HandlerGetChirps(w http.ResponseWriter, r *http.Request) {
	authorID := r.URL.Query().Get("author_id")
	sort := r.URL.Query().Get("sort")
	
	var dbChirps []database.Chirp
	var err error
	
	if authorID != "" {
		parsedAuthorID, err := uuid.Parse(authorID)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid author ID")
			return
		}
		dbChirps, err = cfg.DB.GetChirpsByUserID(r.Context(), parsedAuthorID)
	} else {
		dbChirps, err = cfg.DB.GetAllChirps(r.Context())
	}
	
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirps")
		return
	}

	if sort == "desc" {
		for i, j := 0, len(dbChirps)-1; i < j; i, j = i+1, j-1 {
			dbChirps[i], dbChirps[j] = dbChirps[j], dbChirps[i]
		}
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID.String(),
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID.String(),
		})
	}

	RespondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *ChirpConfig) HandlerGetChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	chirpID := r.PathValue("chirpID")
	parsedID, err := uuid.Parse(chirpID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	dbChirp, err := cfg.DB.GetChirp(r.Context(), parsedID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	RespondWithJSON(w, http.StatusOK, Chirp{
		ID:        dbChirp.ID.String(),
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID.String(),
	})
}

func (cfg *ChirpConfig) HandlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	chirpID := r.PathValue("chirpID")
	parsedChirpID, err := uuid.Parse(chirpID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	dbChirp, err := cfg.DB.GetChirp(r.Context(), parsedChirpID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	if dbChirp.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = cfg.DB.DeleteChirp(r.Context(), parsedChirpID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func cleanChirp(chirp string) string {
	badWords := []string{"fornax", "sharbert", "kerfuffle"}
	for _, word := range badWords {
		re := regexp.MustCompile("(?i)" + word)
		chirp = re.ReplaceAllString(chirp, "****")
	}
	return chirp
}