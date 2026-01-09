package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for mobile app access
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Device ID identified after handshake
	DeviceID string
}

// BaseMessage is the basic message structure for routing
type BaseMessage struct {
	Type     string `json:"type"`
	DeviceID string `json:"deviceId,omitempty"`
	MsgID    string `json:"msgId,omitempty"`
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS error: %v", err)
			}
			break
		}

		// Handle basic protocol messages here
		var msg BaseMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			// 1. DEVICE_IDENTIFY Handshake
			if msg.Type == "DEVICE_IDENTIFY" && msg.DeviceID != "" {
				c.DeviceID = msg.DeviceID
				c.hub.register <- c

				// Send ACK
				ack := map[string]string{
					"type":   "ACK",
					"msgId":  msg.MsgID,
					"status": "connected",
				}
				c.SendJSON(ack)
				continue
			}
		}

		// Broadcast other messages (like SCAN) to all connected clients (Admin UI)
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON sends a JSON message to the client
func (c *Client) SendJSON(v interface{}) error {
	msg, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.send <- msg
	return nil
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Generate temporary ID for web clients until they identify or just stay anonymous listeners
	clientID := "web_" + uuid.New().String()
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), DeviceID: clientID}
	// Register immediately for Web Clients to receive broadcasts
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
