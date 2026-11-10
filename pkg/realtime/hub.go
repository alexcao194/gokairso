package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Event struct {
	Type        string `json:"type"`
	WorkspaceID string `json:"workspaceId"`
	EntityID    string `json:"entityId,omitempty"`
	ActorID     string `json:"actorId,omitempty"`
	ActorConnID string `json:"-"`
	Entity      any    `json:"entity,omitempty"`
}

type Broadcaster interface {
	Broadcast(event Event)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = (pongWait * 9) / 10
	sendBufferSize = 32
)

type conn struct {
	ws     *websocket.Conn
	send   chan []byte
	rooms  []string
	connID string
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*conn]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*conn]struct{})}
}

func (h *Hub) Broadcast(event Event) {
	if event.WorkspaceID == "" {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*conn, 0, len(h.rooms[event.WorkspaceID]))
	for c := range h.rooms[event.WorkspaceID] {
		if event.ActorConnID != "" && c.connID == event.ActorConnID {
			continue
		}
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, workspaceIDs []string, connID string) error {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	c := &conn{ws: ws, send: make(chan []byte, sendBufferSize), rooms: workspaceIDs, connID: connID}
	h.register(c)
	go c.writePump()
	c.readPump(h)
	return nil
}

func (h *Hub) register(c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range c.rooms {
		if h.rooms[room] == nil {
			h.rooms[room] = make(map[*conn]struct{})
		}
		h.rooms[room][c] = struct{}{}
	}
}

func (h *Hub) unregister(c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range c.rooms {
		delete(h.rooms[room], c)
		if len(h.rooms[room]) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (c *conn) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *conn) readPump(h *Hub) {
	defer func() {
		h.unregister(c)
		close(c.send)
		c.ws.Close()
	}()
	c.ws.SetReadLimit(512)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.ws.ReadMessage(); err != nil {
			return
		}
	}
}
