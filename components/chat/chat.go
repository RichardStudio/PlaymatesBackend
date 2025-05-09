package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gofiber/contrib/websocket"
	"playmates/components/repository"
	"playmates/components/service/models"
	"sync"
)

type ConnectionManager struct {
	connections map[int]*websocket.Conn
	mu          sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
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

func HandleWebSocket(c *websocket.Conn, db *sql.DB, connectionManager *ConnectionManager, userID int) {
	if userID <= 0 {
		c.WriteMessage(websocket.CloseMessage, []byte("Invalid user ID"))
		return
	}

	// Добавляем соединение в менеджер
	connectionManager.Add(userID, c)
	fmt.Println("ws new conn")
	defer connectionManager.Remove(userID)

	for {
		// Чтение сообщения
		_, msg, err := c.ReadMessage()
		if err != nil {
			fmt.Println("WebSocket read error:", err)
			c.WriteMessage(websocket.CloseMessage, []byte("Error occurred during reading"))
			break
		}

		// Парсим сообщение
		var message models.Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}

		// Сохраняем сообщение в базе данных
		err = repository.PostMessage(message.Msg, userID, message.ReceiverID, db)
		if err != nil {
			fmt.Println("Error saving message:", err)
			continue
		}

		// Проверяем, что получатель существует
		receiverConn, exists := connectionManager.Get(message.ReceiverID)
		if !exists {
			fmt.Println("Receiver not connected")
			continue
		}

		// Отправляем сообщение получателю
		err = receiverConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			fmt.Println("Error sending message:", err)
			continue
		}
	}
}
