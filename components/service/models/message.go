package models

import "time"

type Message struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Msg        string    `json:"msg"`
	Time       time.Time `json:"time"`
}

type ChatPreview struct {
	LastMessageID   int       `json:"last_message_id"`
	SenderID        int       `json:"sender_id"`
	ReceiverID      int       `json:"receiver_id"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	OtherUserID     int       `json:"other_user_id"`
	OtherUsername   string    `json:"other_username"`
}
