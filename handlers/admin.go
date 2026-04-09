package handlers

import (
	"fmt"
	"net/http"
	"github.com/Kyaromon/Chirpy/internal/database"
	"sync/atomic"
)

type AdminConfig struct {
	DB             *database.Queries
	Platform       string
	FileserverHits *atomic.Int32
}

func (cfg *AdminConfig) HandlerAdminMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.FileserverHits.Load()
	html := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
	w.Write([]byte(html))
}

func (cfg *AdminConfig) HandlerAdminReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if cfg.Platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to reset database")
		return
	}

	cfg.FileserverHits.Store(0)
	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Database reset"})
}