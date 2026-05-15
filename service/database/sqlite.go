package database

import (
	"database/sql"
	_ "embed"
	"errors"
	"net/url"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// NewSQLite returns an application database backed by a SQLite file.
func NewSQLite(path string) (AppDatabase, error) {
	// Pragmas set via DSN apply to every connection the pool opens (setting
	// them with a one-shot Exec only affects the single connection Exec
	// happened to use). FK enforcement keeps referential promises; WAL lets
	// concurrent readers and a single writer coexist; busy_timeout makes the
	// driver wait up to 5 s for a lock rather than immediately erroring —
	// which is the class of failure that showed up as sporadic 500s on login.
	dsn := buildSQLiteDSN(path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := migrateSQLite(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	db := newInMemoryDatabase()
	db.sqlDB = sqlDB
	if err := db.loadUsersFromSQLite(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.loadConversationsFromSQLite(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.loadConversationReadsFromSQLite(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return db, nil
}

// buildSQLiteDSN appends the per-connection pragmas go-sqlite3 recognises as
// query parameters. Bare filenames like ":memory:" or "foo.db" are passed
// through unchanged except for the appended parameters; we intentionally do
// not try to reinterpret them as URIs to avoid corrupting relative paths on
// Windows-style backslashes.
func buildSQLiteDSN(path string) string {
	params := url.Values{}
	params.Set("_foreign_keys", "on")
	params.Set("_journal_mode", "WAL")
	params.Set("_busy_timeout", "5000")
	params.Set("_synchronous", "NORMAL")
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + params.Encode()
}

func migrateSQLite(sqlDB *sql.DB) error {
	if err := ensureSQLiteColumn(sqlDB, "messages", "is_forwarded", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(sqlDB, "messages", "forwarded_from_name", "TEXT"); err != nil {
		return err
	}
	return normalizeStoredReactionsSQLite(sqlDB)
}

func normalizeStoredReactionsSQLite(sqlDB *sql.DB) error {
	_, err := sqlDB.Exec(`
		UPDATE message_reactions
		SET emoticon = CASE emoticon
			WHEN ':thumbsup:' THEN '👍'
			WHEN '+1' THEN '👍'
			WHEN ':heart:' THEN '❤️'
			WHEN '<3' THEN '❤️'
			WHEN '❤' THEN '❤️'
			WHEN ':ok:' THEN '👌'
			WHEN 'OK' THEN '👌'
			WHEN 'ok' THEN '👌'
			WHEN ':laugh:' THEN '😂'
			WHEN ':wow:' THEN '😮'
			WHEN ':pray:' THEN '🙏'
			WHEN ':fire:' THEN '🔥'
			WHEN ':party:' THEN '🎉'
			WHEN ':sad:' THEN '😢'
			WHEN ':eyes:' THEN '👀'
			WHEN ':clap:' THEN '👏'
			WHEN ':star:' THEN '⭐'
			ELSE emoticon
		END
	`)
	return err
}

func ensureSQLiteColumn(sqlDB *sql.DB, table string, column string, definition string) error {
	rows, err := sqlDB.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = sqlDB.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func (db *appdbimpl) loadUsersFromSQLite() error {
	rows, err := db.sqlDB.Query("SELECT id, name, photo FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		var photo sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &photo); err != nil {
			return err
		}
		if photo.Valid {
			user.Photo = &photo.String
		}

		db.usersByID[user.ID] = user
		db.userIDByName[user.Name] = user.ID
	}

	return rows.Err()
}

func (db *appdbimpl) loadConversationsFromSQLite() error {
	rows, err := db.sqlDB.Query("SELECT id, type, name, photo, created_at FROM conversations")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var conversation Conversation
		var photo sql.NullString
		var createdAt string
		if err := rows.Scan(&conversation.ID, &conversation.Type, &conversation.Name, &photo, &createdAt); err != nil {
			return err
		}
		if photo.Valid {
			conversation.Photo = &photo.String
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return err
		}
		conversation.CreatedAt = parsedCreatedAt

		members, err := db.loadConversationMembersFromSQLite(conversation.ID)
		if err != nil {
			return err
		}
		conversation.Members = members

		messages, err := db.loadMessagesFromSQLite(conversation.ID)
		if err != nil {
			return err
		}
		conversation.Messages = messages

		db.conversationsByID[conversation.ID] = conversation
		if conversation.Type == "private" && len(members) == 2 {
			db.privateConversationIDByPair[privateConversationPairKey(members[0].ID, members[1].ID)] = conversation.ID
		}
	}

	return rows.Err()
}

func (db *appdbimpl) loadConversationMembersFromSQLite(conversationID string) ([]User, error) {
	rows, err := db.sqlDB.Query(`
		SELECT user_id
		FROM conversation_members
		WHERE conversation_id = ?
		ORDER BY position ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []User{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		user, ok := db.usersByID[userID]
		if ok {
			members = append(members, user)
		}
	}

	return members, rows.Err()
}

func (db *appdbimpl) loadMessagesFromSQLite(conversationID string) ([]Message, error) {
	rows, err := db.sqlDB.Query(`
		SELECT id, sender_id, timestamp, type, text, photo, reply_to_message_id, is_forwarded, forwarded_from_name
		FROM messages
		WHERE conversation_id = ?
		ORDER BY timestamp ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var message Message
		var senderID string
		var timestamp string
		var text sql.NullString
		var photo sql.NullString
		var replyToMessageID sql.NullString
		var isForwarded int
		var forwardedFromName sql.NullString
		if err := rows.Scan(&message.ID, &senderID, &timestamp, &message.Type, &text, &photo, &replyToMessageID, &isForwarded, &forwardedFromName); err != nil {
			return nil, err
		}

		sender, ok := db.usersByID[senderID]
		if !ok {
			continue
		}
		parsedTimestamp, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return nil, err
		}

		message.ConversationID = conversationID
		message.Sender = sender
		message.Timestamp = parsedTimestamp
		if text.Valid {
			message.Text = &text.String
		}
		if photo.Valid {
			message.Photo = &photo.String
		}
		if replyToMessageID.Valid {
			message.ReplyToMessageID = &replyToMessageID.String
		}
		message.IsForwarded = isForwarded != 0
		if forwardedFromName.Valid {
			message.ForwardedFrom = forwardedFromName.String
		}

		reactions, err := db.loadMessageReactionsFromSQLite(message.ID)
		if err != nil {
			return nil, err
		}
		message.Reactions = reactions

		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (db *appdbimpl) loadMessageReactionsFromSQLite(messageID string) ([]MessageReaction, error) {
	rows, err := db.sqlDB.Query(`
		SELECT user_id, emoticon
		FROM message_reactions
		WHERE message_id = ?
		ORDER BY user_id ASC
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reactions := []MessageReaction{}
	for rows.Next() {
		var userID string
		var emoticon string
		if err := rows.Scan(&userID, &emoticon); err != nil {
			return nil, err
		}

		user, ok := db.usersByID[userID]
		if ok {
			reactions = append(reactions, MessageReaction{
				User:     user,
				Emoticon: normalizeReaction(emoticon),
			})
		}
	}

	return reactions, rows.Err()
}

// upsertUserSQLite mirrors a User into the users table. The unique index on
// `name` is what guarantees rename collisions surface as ErrUserNameTaken.
func (db *appdbimpl) upsertUserSQLite(user User) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec(
		`INSERT INTO users (id, name, photo) VALUES (?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name = excluded.name, photo = excluded.photo`,
		user.ID,
		user.Name,
		nullableString(user.Photo),
	)
	return err
}

func (db *appdbimpl) insertConversationSQLite(conversation Conversation) (err error) {
	if db.sqlDB == nil {
		return nil
	}

	tx, err := db.sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err == nil {
			err = rollbackErr
		}
	}()

	if _, err := tx.Exec(
		"INSERT INTO conversations (id, type, name, photo, created_at) VALUES (?, ?, ?, ?, ?)",
		conversation.ID,
		conversation.Type,
		conversation.Name,
		nullableString(conversation.Photo),
		conversation.CreatedAt.Format(time.RFC3339Nano),
	); err != nil {
		return err
	}

	for index, member := range conversation.Members {
		if _, err := tx.Exec(
			"INSERT INTO conversation_members (conversation_id, user_id, position) VALUES (?, ?, ?)",
			conversation.ID,
			member.ID,
			index,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *appdbimpl) updateGroupNameSQLite(groupID string, name string) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec("UPDATE conversations SET name = ? WHERE id = ?", name, groupID)
	return err
}

func (db *appdbimpl) updateGroupPhotoSQLite(groupID string, photo string) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec("UPDATE conversations SET photo = ? WHERE id = ?", photo, groupID)
	return err
}

func (db *appdbimpl) addConversationMemberSQLite(groupID string, userID string, position int) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec(
		"INSERT OR IGNORE INTO conversation_members (conversation_id, user_id, position) VALUES (?, ?, ?)",
		groupID,
		userID,
		position,
	)
	return err
}

func (db *appdbimpl) removeConversationMemberSQLite(groupID string, userID string) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec("DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?", groupID, userID)
	return err
}

func (db *appdbimpl) insertMessageSQLite(message Message) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec(
		`INSERT INTO messages
			(id, conversation_id, sender_id, timestamp, type, text, photo, reply_to_message_id, is_forwarded, forwarded_from_name)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.ID,
		message.ConversationID,
		message.Sender.ID,
		message.Timestamp.Format(time.RFC3339Nano),
		message.Type,
		nullableString(message.Text),
		nullableString(message.Photo),
		nullableString(message.ReplyToMessageID),
		boolToInt(message.IsForwarded),
		nullableForwardedFrom(message.ForwardedFrom),
	)
	return err
}

func (db *appdbimpl) deleteMessageSQLite(messageID string) (err error) {
	if db.sqlDB == nil {
		return nil
	}

	tx, err := db.sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err == nil {
			err = rollbackErr
		}
	}()

	if _, err := tx.Exec("UPDATE messages SET reply_to_message_id = NULL WHERE reply_to_message_id = ?", messageID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM messages WHERE id = ?", messageID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *appdbimpl) upsertMessageReactionSQLite(messageID string, reaction MessageReaction) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec(
		`INSERT INTO message_reactions (message_id, user_id, emoticon)
			VALUES (?, ?, ?)
			ON CONFLICT(message_id, user_id) DO UPDATE SET emoticon = excluded.emoticon`,
		messageID,
		reaction.User.ID,
		reaction.Emoticon,
	)
	return err
}

func (db *appdbimpl) deleteMessageReactionSQLite(messageID string, userID string) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec("DELETE FROM message_reactions WHERE message_id = ? AND user_id = ?", messageID, userID)
	return err
}

func (db *appdbimpl) loadConversationReadsFromSQLite() error {
	rows, err := db.sqlDB.Query("SELECT conversation_id, user_id, last_read_at FROM conversation_reads")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var conversationID string
		var userID string
		var lastReadAt string
		if err := rows.Scan(&conversationID, &userID, &lastReadAt); err != nil {
			return err
		}
		parsed, err := time.Parse(time.RFC3339Nano, lastReadAt)
		if err != nil {
			return err
		}
		reads, ok := db.readsByConversation[conversationID]
		if !ok {
			reads = map[string]time.Time{}
			db.readsByConversation[conversationID] = reads
		}
		reads[userID] = parsed
	}
	return rows.Err()
}

func (db *appdbimpl) upsertConversationReadSQLite(conversationID string, userID string, at time.Time) error {
	if db.sqlDB == nil {
		return nil
	}
	_, err := db.sqlDB.Exec(
		`INSERT INTO conversation_reads (conversation_id, user_id, last_read_at)
			VALUES (?, ?, ?)
			ON CONFLICT(conversation_id, user_id) DO UPDATE SET last_read_at = excluded.last_read_at`,
		conversationID,
		userID,
		at.Format(time.RFC3339Nano),
	)
	return err
}

func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableForwardedFrom(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
