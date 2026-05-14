package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"playmates/components/connection-manager"
	"playmates/components/playmates/models"
	"playmates/components/repository"
	"playmates/components/sealer"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db                *sql.DB
	jwtSecret         string
	repo              *repository.Repository
	connectionManager *connection_manager.ConnectionManager
	sealer            *sealer.Sealer
}

func New(db *sql.DB, jwtSecret string, repository *repository.Repository, connManager *connection_manager.ConnectionManager, sealer *sealer.Sealer) *Service {
	return &Service{
		db:                db,
		jwtSecret:         jwtSecret,
		repo:              repository,
		connectionManager: connManager,
		sealer:            sealer,
	}
}

func (s *Service) GetIdFromToken(tokenString string) (int, error) {
	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return -1, fmt.Errorf("failed to parse token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return -1, fmt.Errorf("failed to parse claims")
	}

	UserIdFloat, ok := claims["user_id"].(float64)
	if !ok {
		return -1, fmt.Errorf("no user id found")
	}

	UserId := int(UserIdFloat)

	return UserId, nil
}

func (s *Service) HandleWebSocket(c *websocket.Conn, userID int) {
	if userID <= 0 {
		c.WriteMessage(websocket.CloseMessage, []byte("Invalid user ID"))
		return
	}

	// Добавляем соединение в менеджер
	s.connectionManager.Add(userID, c)
	defer s.connectionManager.Remove(userID)

	for {
		// Чтение сообщения
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			c.WriteMessage(websocket.CloseMessage, []byte("Error occurred during reading"))
			break
		}

		// Парсим сообщение
		var message models.Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		sealedMsg, err := s.sealer.Encrypt([]byte(message.Msg))
		if err != nil {
			log.Println("Error encrypting message:", err)
			continue
		}

		// Сохраняем сообщение в базе данных
		err = s.repo.PostMessage(string(sealedMsg), userID, message.ReceiverID)
		if err != nil {
			log.Println("Error saving message:", err)
			continue
		}

		// Проверяем, что получатель существует
		receiverConn, exists := s.connectionManager.Get(message.ReceiverID)
		if !exists {
			continue
		}

		// Отправляем сообщение получателю
		err = receiverConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error sending message:", err)
			continue
		}
	}
}

func (s *Service) Register(username, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	err = s.repo.Register(username, email, string(hashedPassword))

	return err
}

func (s *Service) Login(email, password string) (string, error) {
	email = strings.ToLower(email)

	user, err := s.repo.Login(email)
	if err != nil {
		log.Printf("err login email: %s, err: %w\n", email, err)
		return "", fmt.Errorf("invalid email or password")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"user_id":  user.ID,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt: %w", err)
	}

	return jwtString, nil
}

func (s *Service) GetUser(userID int) (models.User, error) {
	user, err := s.repo.GetUser(userID)
	if err != nil {
		log.Printf("err get user: %d, err: %v\n", userID, err)
		return models.User{}, err
	}

	return user, nil
}

func (s *Service) SetUser(user models.User) error {
	err := s.repo.SetUser(user)
	if err != nil {
		log.Printf("err set user: %d, err: %v\n", user.ID, err)
		return err
	}

	return nil
}

func (s *Service) SearchUsers(minAge, maxAge, offset int, games []string, gender string) ([]models.User, int, error) {
	for i, _ := range games {
		games[i] = strings.ToLower(games[i])
	}

	users, total, err := s.repo.SearchUsers(minAge, maxAge, offset, games, gender)
	if err != nil {
		log.Printf("err search users: %v\n", err)
		return nil, 0, err
	}

	return users, total, nil
}

func (s *Service) GetMessages(currentUserID, otherUserID int) ([]models.Message, error) {
	msgsDB, err := s.repo.GetMessages(currentUserID, otherUserID)
	if err != nil {
		log.Printf("err get messages: %v\n", err)
		return nil, err
	}

	msgs := make([]models.Message, 0, len(msgsDB))

	for i, msg := range msgsDB {
		decryptedMsg, err := s.sealer.Decrypt(msg.Msg)
		if err != nil {
			log.Printf("err decrypt message: %v\n", err)
			return nil, err
		}

		msgs[i] = models.Message{
			ID:         msg.ID,
			SenderID:   msg.SenderID,
			ReceiverID: msg.ReceiverID,
			Msg:        string(decryptedMsg),
			Time:       msg.Time,
		}
	}

	return msgs, nil
}

func (s *Service) GetUserChats(userID int) ([]models.ChatPreview, error) {
	chatsDB, err := s.repo.GetUserChats(userID)
	if err != nil {
		log.Printf("err get user chats: %v\n", err)
		return nil, err
	}

	chats := make([]models.ChatPreview, 0, len(chatsDB))

	for i, chat := range chatsDB {
		decryptedLastMsg, err := s.sealer.Decrypt(chat.LastMessage)
		if err != nil {
			log.Printf("err decrypt last message: %v\n", err)
			return nil, err
		}

		chats[i] = models.ChatPreview{
			LastMessageID:   chat.LastMessageID,
			SenderID:        chat.SenderID,
			ReceiverID:      chat.ReceiverID,
			LastMessage:     string(decryptedLastMsg),
			LastMessageTime: chat.LastMessageTime,
			OtherUserID:     chat.OtherUserID,
			OtherUsername:   chat.OtherUsername,
		}
	}

	return chats, nil
}

func (s *Service) ParseToken(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	return token, err
}
