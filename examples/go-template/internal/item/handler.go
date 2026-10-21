package item

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Item é o endpoint de exemplo que o "crudding add endpoint" copia.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	mu    sync.Mutex
	items []Item
)

// Register expõe GET e POST /items.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /items", list)
	mux.HandleFunc("POST /items", create)
}

func list(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	writeJSON(w, http.StatusOK, items)
}

func create(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	item.ID = len(items) + 1
	items = append(items, item)
	mu.Unlock()
	writeJSON(w, http.StatusCreated, item)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
