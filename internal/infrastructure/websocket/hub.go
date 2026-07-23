package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/realtime/model"
	realtimeService "github.com/gurkanfikretgunak/masterfabric-go/internal/domain/realtime/service"
)

// Hub is an in-memory WebSocket connection manager.
type Hub struct {
	logger         *slog.Logger
	maxConnections int
	mu             sync.RWMutex
	clients        map[string]*client
	rooms          map[model.RoomKey]map[string]*client
	orgRooms       map[uuid.UUID]map[model.RoomKey]struct{}
	closed         bool
}

// NewHub creates a new in-memory Hub.
func NewHub(logger *slog.Logger, maxConnections int) *Hub {
	if maxConnections <= 0 {
		maxConnections = 1000
	}
	return &Hub{
		logger:         logger,
		maxConnections: maxConnections,
		clients:        make(map[string]*client),
		rooms:          make(map[model.RoomKey]map[string]*client),
		orgRooms:       make(map[uuid.UUID]map[model.RoomKey]struct{}),
	}
}

// Register adds a client and subscribes it to the default events channel.
func (h *Hub) Register(info realtimeService.ClientInfo, send chan []byte) (unregister func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return func() {}
	}
	if len(h.clients) >= h.maxConnections {
		close(send)
		return func() {}
	}

	c := &client{
		id:   info.ID,
		info: info,
		send: send,
		hub:  h,
	}
	h.clients[c.id] = c

	defaultRoom, err := model.BuildRoomKey(info.OrganizationID, info.AppID, model.DefaultChannel)
	if err == nil {
		h.addToRoomLocked(c, defaultRoom)
	}

	return func() { h.removeClient(c) }
}

// Subscribe adds a client to an additional channel room.
func (h *Hub) Subscribe(clientID, channel string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, ok := h.clients[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}
	room, err := model.BuildRoomKey(c.info.OrganizationID, c.info.AppID, channel)
	if err != nil {
		return err
	}
	h.addToRoomLocked(c, room)
	return nil
}

// Unsubscribe removes a client from a channel room.
func (h *Hub) Unsubscribe(clientID, channel string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, ok := h.clients[clientID]
	if !ok {
		return fmt.Errorf("client not found")
	}
	room, err := model.BuildRoomKey(c.info.OrganizationID, c.info.AppID, channel)
	if err != nil {
		return err
	}
	h.removeFromRoomLocked(c, room)
	return nil
}

// SendToClient delivers a message to a single connected client.
func (h *Hub) SendToClient(clientID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[clientID]
	if !ok || h.closed {
		return
	}
	select {
	case c.send <- payload:
	default:
		if h.logger != nil {
			h.logger.Warn("websocket client send buffer full", "client_id", clientID)
		}
	}
}

// Broadcast sends a message to all clients subscribed to a room.
func (h *Hub) Broadcast(room model.RoomKey, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.broadcastLocked(room, payload)
}

// BroadcastToOrganization sends a message to all rooms for an organization and channel.
func (h *Hub) BroadcastToOrganization(orgID uuid.UUID, channel string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	orgMap, ok := h.orgRooms[orgID]
	if !ok || h.closed {
		return
	}

	suffix := ":channel:" + channel
	for room := range orgMap {
		if strings.HasSuffix(string(room), suffix) {
			h.broadcastLocked(room, payload)
		}
	}
}

func (h *Hub) broadcastLocked(room model.RoomKey, payload []byte) {
	if h.closed {
		return
	}
	members, ok := h.rooms[room]
	if !ok {
		return
	}
	for _, c := range members {
		select {
		case c.send <- payload:
		default:
			if h.logger != nil {
				h.logger.Warn("websocket client send buffer full, dropping message", "client_id", c.id)
			}
		}
	}
}

// ConnectionCount returns the number of active connections.
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Close shuts down the hub and disconnects all clients.
func (h *Hub) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true
	for _, c := range h.clients {
		c.closeSend()
	}
	h.clients = make(map[string]*client)
	h.rooms = make(map[model.RoomKey]map[string]*client)
	h.orgRooms = make(map[uuid.UUID]map[model.RoomKey]struct{})
	return nil
}

func (h *Hub) addToRoomLocked(c *client, room model.RoomKey) {
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[string]*client)
	}
	h.rooms[room][c.id] = c
	c.subscribe(room)

	if c.info.OrganizationID != uuid.Nil {
		if h.orgRooms[c.info.OrganizationID] == nil {
			h.orgRooms[c.info.OrganizationID] = make(map[model.RoomKey]struct{})
		}
		h.orgRooms[c.info.OrganizationID][room] = struct{}{}
	}
}

func (h *Hub) removeFromRoomLocked(c *client, room model.RoomKey) {
	if members, ok := h.rooms[room]; ok {
		delete(members, c.id)
		if len(members) == 0 {
			delete(h.rooms, room)
			if c.info.OrganizationID != uuid.Nil {
				if orgMap, found := h.orgRooms[c.info.OrganizationID]; found {
					delete(orgMap, room)
					if len(orgMap) == 0 {
						delete(h.orgRooms, c.info.OrganizationID)
					}
				}
			}
		}
	}
	c.unsubscribe(room)
}

func (h *Hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[c.id]; !ok {
		return
	}
	delete(h.clients, c.id)
	for room := range c.rooms {
		if members, ok := h.rooms[room]; ok {
			delete(members, c.id)
			if len(members) == 0 {
				delete(h.rooms, room)
				if c.info.OrganizationID != uuid.Nil {
					if orgMap, found := h.orgRooms[c.info.OrganizationID]; found {
						delete(orgMap, room)
						if len(orgMap) == 0 {
							delete(h.orgRooms, c.info.OrganizationID)
						}
					}
				}
			}
		}
	}
	c.closeSend()
	if h.logger != nil {
		h.logger.Info("websocket client disconnected",
			"client_id", c.id,
			"user_id", c.info.UserID,
			"org_id", c.info.OrganizationID,
			"app_id", c.info.AppID,
		)
	}
}

// GetClientInfo returns client info for a connected client.
func (h *Hub) GetClientInfo(clientID string) (realtimeService.ClientInfo, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[clientID]
	if !ok {
		return realtimeService.ClientInfo{}, false
	}
	return c.info, true
}

// NewClientID generates a unique client identifier.
func NewClientID() string {
	return uuid.New().String()
}
