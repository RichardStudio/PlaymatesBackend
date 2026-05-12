package connection_manager

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type ConnectionManager struct {
	connections map[int]*websocket.Conn
	mu          sync.Mutex
}

func New() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[int]*websocket.Conn),
	}
}

func (cm *ConnectionManager) Add(userID int, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Если соединение уже существует, закрываем его
	if existingConn, exists := cm.connections[userID]; exists {
		existingConn.Close()
	}

	cm.connections[userID] = conn
}

func (cm *ConnectionManager) Remove(userID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, userID)
}

func (cm *ConnectionManager) Get(userID int) (*websocket.Conn, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	conn, exists := cm.connections[userID]
	return conn, exists
}
