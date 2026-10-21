package health

import (
	"encoding/json"
	"net/http"
)

// Register expõe GET /health.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "go-template"})
	})
}
