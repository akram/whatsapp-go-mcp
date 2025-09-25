package models

import (
	"database/sql"
	"time"
)

// Message represents a WhatsApp message stored in the database
type Message struct {
	ID        int64     `json:"id"`
	Time      time.Time `json:"time"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	IsFromMe  bool      `json:"is_from_me"`
	MediaType string    `json:"media_type"`
	Filename  string    `json:"filename"`
	ChatJID   string    `json:"chat_jid"`
	MessageID string    `json:"message_id"`
}

// Contact represents a WhatsApp contact
type Contact struct {
	JID       string `json:"jid"`
	Name      string `json:"name"`
	PushName  string `json:"push_name"`
	IsGroup   bool   `json:"is_group"`
	IsBlocked bool   `json:"is_blocked"`
}

// Chat represents a WhatsApp chat/conversation
type Chat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	UnreadCount     int       `json:"unread_count"`
	IsGroup         bool      `json:"is_group"`
	IsArchived      bool      `json:"is_archived"`
	IsMuted         bool      `json:"is_muted"`
}

// Database represents the database connection and operations
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.initTables(); err != nil {
		return nil, err
	}

	return database, nil
}

// initTables creates the necessary tables
func (d *Database) initTables() error {
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		time DATETIME NOT NULL,
		sender TEXT NOT NULL,
		content TEXT,
		is_from_me BOOLEAN NOT NULL,
		media_type TEXT,
		filename TEXT,
		chat_jid TEXT NOT NULL,
		message_id TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createContactsTable := `
	CREATE TABLE IF NOT EXISTS contacts (
		jid TEXT PRIMARY KEY,
		name TEXT,
		push_name TEXT,
		is_group BOOLEAN NOT NULL DEFAULT FALSE,
		is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createChatsTable := `
	CREATE TABLE IF NOT EXISTS chats (
		jid TEXT PRIMARY KEY,
		name TEXT,
		last_message TEXT,
		last_message_time DATETIME,
		unread_count INTEGER DEFAULT 0,
		is_group BOOLEAN NOT NULL DEFAULT FALSE,
		is_archived BOOLEAN NOT NULL DEFAULT FALSE,
		is_muted BOOLEAN NOT NULL DEFAULT FALSE,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create indexes for better performance
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_messages_time ON messages(time);",
		"CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender);",
		"CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid);",
		"CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);",
		"CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);",
		"CREATE INDEX IF NOT EXISTS idx_chats_last_message_time ON chats(last_message_time);",
	}

	queries := []string{createMessagesTable, createContactsTable, createChatsTable}
	queries = append(queries, createIndexes...)

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// StoreMessage stores a message in the database
func (d *Database) StoreMessage(msg *Message) error {
	query := `
	INSERT OR REPLACE INTO messages 
	(time, sender, content, is_from_me, media_type, filename, chat_jid, message_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, msg.Time, msg.Sender, msg.Content, msg.IsFromMe,
		msg.MediaType, msg.Filename, msg.ChatJID, msg.MessageID)
	return err
}

// GetMessages retrieves messages with optional filters
func (d *Database) GetMessages(chatJID string, limit int, offset int) ([]*Message, error) {
	query := `
	SELECT id, time, sender, content, is_from_me, media_type, filename, chat_jid, message_id
	FROM messages 
	WHERE chat_jid = ?
	ORDER BY time DESC
	LIMIT ? OFFSET ?`

	rows, err := d.db.Query(query, chatJID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.Time, &msg.Sender, &msg.Content,
			&msg.IsFromMe, &msg.MediaType, &msg.Filename, &msg.ChatJID, &msg.MessageID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessageByID retrieves a specific message by its ID
func (d *Database) GetMessageByID(messageID string) (*Message, error) {
	query := `
	SELECT id, time, sender, content, is_from_me, media_type, filename, chat_jid, message_id
	FROM messages 
	WHERE message_id = ?`

	msg := &Message{}
	err := d.db.QueryRow(query, messageID).Scan(
		&msg.ID, &msg.Time, &msg.Sender, &msg.Content,
		&msg.IsFromMe, &msg.MediaType, &msg.Filename, &msg.ChatJID, &msg.MessageID)

	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetLastMessageWithContact gets the most recent message with a specific contact
func (d *Database) GetLastMessageWithContact(contactJID string) (*Message, error) {
	query := `
	SELECT id, time, sender, content, is_from_me, media_type, filename, chat_jid, message_id
	FROM messages 
	WHERE sender = ? OR (is_from_me = 1 AND chat_jid = ?)
	ORDER BY time DESC
	LIMIT 1`

	msg := &Message{}
	err := d.db.QueryRow(query, contactJID, contactJID).Scan(
		&msg.ID, &msg.Time, &msg.Sender, &msg.Content,
		&msg.IsFromMe, &msg.MediaType, &msg.Filename, &msg.ChatJID, &msg.MessageID)

	if err != nil {
		return nil, err
	}
	return msg, nil
}

// StoreContact stores or updates a contact
func (d *Database) StoreContact(contact *Contact) error {
	query := `
	INSERT OR REPLACE INTO contacts 
	(jid, name, push_name, is_group, is_blocked)
	VALUES (?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, contact.JID, contact.Name, contact.PushName,
		contact.IsGroup, contact.IsBlocked)
	return err
}

// SearchContacts searches for contacts by name or JID
func (d *Database) SearchContacts(query string) ([]*Contact, error) {
	sqlQuery := `
	SELECT jid, name, push_name, is_group, is_blocked
	FROM contacts 
	WHERE name LIKE ? OR jid LIKE ? OR push_name LIKE ?
	ORDER BY name ASC`

	searchTerm := "%" + query + "%"
	rows, err := d.db.Query(sqlQuery, searchTerm, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		contact := &Contact{}
		err := rows.Scan(&contact.JID, &contact.Name, &contact.PushName,
			&contact.IsGroup, &contact.IsBlocked)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// StoreChat stores or updates a chat
func (d *Database) StoreChat(chat *Chat) error {
	query := `
	INSERT OR REPLACE INTO chats 
	(jid, name, last_message, last_message_time, unread_count, is_group, is_archived, is_muted)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, chat.JID, chat.Name, chat.LastMessage,
		chat.LastMessageTime, chat.UnreadCount, chat.IsGroup, chat.IsArchived, chat.IsMuted)
	return err
}

// GetChats retrieves all chats
func (d *Database) GetChats() ([]*Chat, error) {
	query := `
	SELECT jid, name, last_message, last_message_time, unread_count, is_group, is_archived, is_muted
	FROM chats 
	ORDER BY last_message_time DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*Chat
	for rows.Next() {
		chat := &Chat{}
		err := rows.Scan(&chat.JID, &chat.Name, &chat.LastMessage,
			&chat.LastMessageTime, &chat.UnreadCount, &chat.IsGroup,
			&chat.IsArchived, &chat.IsMuted)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// GetChatByJID retrieves a specific chat by JID
func (d *Database) GetChatByJID(jid string) (*Chat, error) {
	query := `
	SELECT jid, name, last_message, last_message_time, unread_count, is_group, is_archived, is_muted
	FROM chats 
	WHERE jid = ?`

	chat := &Chat{}
	err := d.db.QueryRow(query, jid).Scan(&chat.JID, &chat.Name, &chat.LastMessage,
		&chat.LastMessageTime, &chat.UnreadCount, &chat.IsGroup,
		&chat.IsArchived, &chat.IsMuted)

	if err != nil {
		return nil, err
	}
	return chat, nil
}

// GetChatsByContact retrieves all chats involving a specific contact
func (d *Database) GetChatsByContact(contactJID string) ([]*Chat, error) {
	query := `
	SELECT DISTINCT c.jid, c.name, c.last_message, c.last_message_time, c.unread_count, c.is_group, c.is_archived, c.is_muted
	FROM chats c
	JOIN messages m ON c.jid = m.chat_jid
	WHERE m.sender = ? OR (m.is_from_me = 1 AND c.jid = ?)
	ORDER BY c.last_message_time DESC`

	rows, err := d.db.Query(query, contactJID, contactJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*Chat
	for rows.Next() {
		chat := &Chat{}
		err := rows.Scan(&chat.JID, &chat.Name, &chat.LastMessage,
			&chat.LastMessageTime, &chat.UnreadCount, &chat.IsGroup,
			&chat.IsArchived, &chat.IsMuted)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, nil
}
