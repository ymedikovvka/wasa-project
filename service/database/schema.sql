CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	photo TEXT
);

CREATE TABLE IF NOT EXISTS conversations (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	photo TEXT,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_members (
	conversation_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	position INTEGER NOT NULL,
	PRIMARY KEY (conversation_id, user_id),
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	conversation_id TEXT NOT NULL,
	sender_id TEXT NOT NULL,
	timestamp TEXT NOT NULL,
	type TEXT NOT NULL,
	text TEXT,
	photo TEXT,
	reply_to_message_id TEXT,
	is_forwarded INTEGER NOT NULL DEFAULT 0,
	forwarded_from_name TEXT,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
	FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (reply_to_message_id) REFERENCES messages(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS message_reactions (
	message_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	emoticon TEXT NOT NULL,
	PRIMARY KEY (message_id, user_id),
	FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- conversation_reads tracks the last time each user opened a conversation.
-- The value doubles as a delivery marker: any message older than
-- last_read_at is considered read by that user; any message newer is
-- still in the "received" (delivered) state.
CREATE TABLE IF NOT EXISTS conversation_reads (
	conversation_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	last_read_at TEXT NOT NULL,
	PRIMARY KEY (conversation_id, user_id),
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes on hot paths: message listing by conversation, timeline sorting,
-- per-user conversation lookups, reaction fetches.
CREATE INDEX IF NOT EXISTS idx_messages_conversation_time
	ON messages (conversation_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_conversation_members_user
	ON conversation_members (user_id);
CREATE INDEX IF NOT EXISTS idx_message_reactions_message
	ON message_reactions (message_id);
CREATE INDEX IF NOT EXISTS idx_conversation_reads_user
	ON conversation_reads (user_id);
