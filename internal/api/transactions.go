package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"txpool-viz/internal/transactions"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// TransactionHandler handles transaction-related HTTP endpoints
type TransactionHandler struct {
	streamer *transactions.TransactionStreamer
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(streamer *transactions.TransactionStreamer) *TransactionHandler {
	return &TransactionHandler{
		streamer: streamer,
		clients:  make(map[*websocket.Conn]bool),
	}
}

// RegisterRoutes registers the transaction routes
func (h *TransactionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/transactions/ws", h.handleWebSocket)
	mux.HandleFunc("/api/transactions", h.handleGetTransactions)
	mux.HandleFunc("/api/transactions/", h.handleGetTransaction)
}

// handleWebSocket handles WebSocket connections for real-time transaction updates
func (h *TransactionHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	// Send initial transaction list
	txs := h.streamer.GetAllTransactions()
	if err := conn.WriteJSON(txs); err != nil {
		return
	}

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleGetTransactions returns all transactions with optional filtering
func (h *TransactionHandler) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	txs := h.streamer.GetAllTransactions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

// handleGetTransaction returns a specific transaction by hash
func (h *TransactionHandler) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Path[len("/api/transactions/"):]
	tx, exists := h.streamer.GetTransaction(hash)
	if !exists {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
}

// BroadcastTransaction broadcasts a transaction update to all connected clients
func (h *TransactionHandler) BroadcastTransaction(tx *transactions.StoredTransaction) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if err := client.WriteJSON(tx); err != nil {
			client.Close()
			delete(h.clients, client)
		}
	}
}
