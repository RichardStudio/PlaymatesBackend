package repository

import (
	"fmt"
	"playmates/components/playmates/models"
)

func (r *Repository) PostMessage(message string, senderID, receiverID int) error {
	query := `
        INSERT INTO messages (sender_id, receiver_id, message)
        VALUES ($1, $2, $3)
    `
	_, err := r.db.Exec(query, senderID, receiverID, message)
	if err != nil {
		return fmt.Errorf("error while inserting message into the database: %w", err)
	}

	return nil
}

func (r *Repository) GetMessages(currentUserID, otherUserID int) ([]models.MessageDB, error) {
	query := `
        SELECT sender_id, message, created_at
        FROM messages
        WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
        ORDER BY created_at ASC
    `
	rows, err := r.db.Query(query, currentUserID, otherUserID)
	if err != nil {
		return nil, fmt.Errorf("error while getting messages from the database: %w", err)
	}
	defer rows.Close()

	var messages []models.MessageDB
	for rows.Next() {
		var msg models.MessageDB
		err := rows.Scan(&msg.SenderID, &msg.Msg, &msg.Time)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *Repository) GetUserChats(userID int) ([]models.ChatPreviewDB, error) {
	query := `
        SELECT DISTINCT ON (other_user_id)
            m.id AS message_id,
            m.sender_id,
            m.receiver_id,
            m.message,
            m.created_at,
            CASE
                WHEN m.sender_id = $1 THEN m.receiver_id
                ELSE m.sender_id
            END AS other_user_id,
            u.username AS other_username
        FROM messages m
        JOIN users u ON u.id = CASE
            WHEN m.sender_id = $1 THEN m.receiver_id
            ELSE m.sender_id
        END
        WHERE m.sender_id = $1 OR m.receiver_id = $1
        ORDER BY other_user_id, m.created_at DESC
    `

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error while getting user chats: %w", err)
	}
	defer rows.Close()

	var chats []models.ChatPreviewDB
	for rows.Next() {
		var chat models.ChatPreviewDB
		err := rows.Scan(
			&chat.LastMessageID,
			&chat.SenderID,
			&chat.ReceiverID,
			&chat.LastMessage,
			&chat.LastMessageTime,
			&chat.OtherUserID,
			&chat.OtherUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, nil
}
