package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/outlet-sh/outlet/internal/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now (can be restricted in production)
		return true
	},
}

// Handler creates a WebSocket handler for the given hub
// Browsers cannot set headers on a WebSocket, so the admin access token comes
// in the token query parameter. Connections without a valid token are refused.
func Handler(hub *Hub, accessSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := utils.ValidateToken(r.URL.Query().Get("token"), accessSecret)
		if err != nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims.UserID

		clientID := r.URL.Query().Get("clientId")
		if clientID == "" {
			clientID = "client_" + r.RemoteAddr
		}

		// Upgrade connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create new client
		client := NewClient(hub, conn, clientID, userID)

		// Register client with hub
		hub.register <- client

		// Start goroutines for reading and writing
		go client.WritePump()
		go client.ReadPump()

		log.Printf("WebSocket client connected: %s (user: %s)", clientID, userID)
	}
}
