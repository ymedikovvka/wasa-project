package database

import (
	"path/filepath"
	"testing"
)

func TestSQLitePersistsUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	if _, err := db.SetUserPhoto(medi.ID, "ZmFrZS1pbWFnZQ=="); err != nil {
		t.Fatalf("setting user photo: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	sameUser, err := reopened.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("loading existing user: %v", err)
	}
	if sameUser.ID != medi.ID {
		t.Fatalf("expected persisted user id %q, got %q", medi.ID, sameUser.ID)
	}
	if sameUser.Photo == nil || *sameUser.Photo != "ZmFrZS1pbWFnZQ==" {
		t.Fatalf("expected persisted photo, got %+v", sameUser)
	}
}

func TestSQLitePersistsUserRename(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	renamed, err := db.SetUserName(medi.ID, "Medina")
	if err != nil {
		t.Fatalf("renaming user: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	sameUser, err := reopened.LoginOrCreateUser("Medina")
	if err != nil {
		t.Fatalf("loading renamed user: %v", err)
	}
	if sameUser.ID != renamed.ID {
		t.Fatalf("expected renamed user id %q, got %q", renamed.ID, sameUser.ID)
	}

	_, err = reopened.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("old username should be available after rename: %v", err)
	}
}

func TestSQLitePersistsPrivateConversation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating Medi: %v", err)
	}
	lina, err := db.LoginOrCreateUser("Lina")
	if err != nil {
		t.Fatalf("creating Lina: %v", err)
	}
	conversation, created, err := db.StartPrivateConversation(medi.ID, lina.ID)
	if err != nil {
		t.Fatalf("creating private conversation: %v", err)
	}
	if !created {
		t.Fatal("expected new private conversation")
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	loaded, err := reopened.GetConversationForUser(medi.ID, conversation.ID)
	if err != nil {
		t.Fatalf("loading private conversation: %v", err)
	}
	if loaded.ID != conversation.ID || loaded.Type != "private" || len(loaded.Members) != 2 {
		t.Fatalf("unexpected loaded private conversation: %+v", loaded)
	}

	sameConversation, created, err := reopened.StartPrivateConversation(lina.ID, medi.ID)
	if err != nil {
		t.Fatalf("loading existing private pair: %v", err)
	}
	if created || sameConversation.ID != conversation.ID {
		t.Fatalf("expected existing private conversation %q, got created=%v conversation=%+v", conversation.ID, created, sameConversation)
	}
}

func TestSQLitePersistsGroupConversation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating Medi: %v", err)
	}
	lina, err := db.LoginOrCreateUser("Lina")
	if err != nil {
		t.Fatalf("creating Lina: %v", err)
	}
	nora, err := db.LoginOrCreateUser("Nora")
	if err != nil {
		t.Fatalf("creating Nora: %v", err)
	}

	group, err := db.CreateGroup(medi.ID, "Study Group", []string{lina.ID})
	if err != nil {
		t.Fatalf("creating group: %v", err)
	}
	if _, err := db.SetGroupName(medi.ID, group.ID, "Exam Team"); err != nil {
		t.Fatalf("renaming group: %v", err)
	}
	if _, err := db.SetGroupPhoto(lina.ID, group.ID, "ZmFrZS1pbWFnZQ=="); err != nil {
		t.Fatalf("setting group photo: %v", err)
	}
	if _, err := db.AddToGroup(medi.ID, group.ID, nora.ID); err != nil {
		t.Fatalf("adding Nora: %v", err)
	}
	if err := db.LeaveGroup(lina.ID, group.ID); err != nil {
		t.Fatalf("Lina leaving group: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	loaded, err := reopened.GetConversationForUser(medi.ID, group.ID)
	if err != nil {
		t.Fatalf("loading group: %v", err)
	}
	if loaded.Name != "Exam Team" || loaded.Photo == nil || *loaded.Photo != "ZmFrZS1pbWFnZQ==" {
		t.Fatalf("unexpected loaded group metadata: %+v", loaded)
	}
	if len(loaded.Members) != 2 {
		t.Fatalf("expected 2 remaining members, got %+v", loaded.Members)
	}
	members := map[string]bool{}
	for _, member := range loaded.Members {
		members[member.ID] = true
	}
	if !members[medi.ID] || !members[nora.ID] || members[lina.ID] {
		t.Fatalf("unexpected loaded group members: %+v", loaded.Members)
	}

	_, err = reopened.GetConversationForUser(lina.ID, group.ID)
	if err != ErrConversationForbidden {
		t.Fatalf("expected Lina to be outside the group, got %v", err)
	}
}

func TestSQLitePersistsMessagesAndReactions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating Medi: %v", err)
	}
	lina, err := db.LoginOrCreateUser("Lina")
	if err != nil {
		t.Fatalf("creating Lina: %v", err)
	}
	conversation, _, err := db.StartPrivateConversation(medi.ID, lina.ID)
	if err != nil {
		t.Fatalf("creating private conversation: %v", err)
	}

	questionText := "Question"
	question, err := db.SendMessage(medi.ID, conversation.ID, "text", &questionText, nil, nil)
	if err != nil {
		t.Fatalf("sending question: %v", err)
	}
	answerText := "Answer"
	replyID := question.ID
	answer, err := db.SendMessage(lina.ID, conversation.ID, "text", &answerText, nil, &replyID)
	if err != nil {
		t.Fatalf("sending answer: %v", err)
	}
	reaction, err := db.CommentMessage(lina.ID, question.ID, ":thumbsup:")
	if err != nil {
		t.Fatalf("commenting message: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	loaded, err := reopened.GetConversationForUser(medi.ID, conversation.ID)
	if err != nil {
		t.Fatalf("loading conversation: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[0].ID != question.ID || loaded.Messages[0].Text == nil || *loaded.Messages[0].Text != questionText {
		t.Fatalf("unexpected loaded question: %+v", loaded.Messages[0])
	}
	if len(loaded.Messages[0].Reactions) != 1 || loaded.Messages[0].Reactions[0].User.ID != reaction.User.ID || loaded.Messages[0].Reactions[0].Emoticon != reaction.Emoticon {
		t.Fatalf("unexpected loaded reaction: %+v", loaded.Messages[0].Reactions)
	}
	if loaded.Messages[1].ID != answer.ID || loaded.Messages[1].ReplyToMessageID == nil || *loaded.Messages[1].ReplyToMessageID != question.ID {
		t.Fatalf("unexpected loaded answer reply: %+v", loaded.Messages[1])
	}
}

func TestSQLiteNormalizesLegacyReactionsOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}
	impl := db.(*appdbimpl)

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating Medi: %v", err)
	}
	lina, err := db.LoginOrCreateUser("Lina")
	if err != nil {
		t.Fatalf("creating Lina: %v", err)
	}
	conversation, _, err := db.StartPrivateConversation(medi.ID, lina.ID)
	if err != nil {
		t.Fatalf("creating private conversation: %v", err)
	}
	text := "Legacy heart"
	message, err := db.SendMessage(medi.ID, conversation.ID, "text", &text, nil, nil)
	if err != nil {
		t.Fatalf("sending message: %v", err)
	}
	if _, err := db.CommentMessage(lina.ID, message.ID, "👍"); err != nil {
		t.Fatalf("commenting message: %v", err)
	}
	if _, err := impl.sqlDB.Exec("UPDATE message_reactions SET emoticon = ? WHERE message_id = ? AND user_id = ?", "<3", message.ID, lina.ID); err != nil {
		t.Fatalf("seeding legacy reaction: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}
	loaded, err := reopened.GetConversationForUser(medi.ID, conversation.ID)
	if err != nil {
		t.Fatalf("loading conversation: %v", err)
	}
	if len(loaded.Messages) != 1 || len(loaded.Messages[0].Reactions) != 1 || loaded.Messages[0].Reactions[0].Emoticon != "❤️" {
		t.Fatalf("expected canonical heart reaction, got %+v", loaded.Messages)
	}
}

func TestSQLitePersistsForwardDeleteAndUncomment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wasatext-test.db")

	db, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("creating sqlite database: %v", err)
	}

	medi, err := db.LoginOrCreateUser("Medi")
	if err != nil {
		t.Fatalf("creating Medi: %v", err)
	}
	lina, err := db.LoginOrCreateUser("Lina")
	if err != nil {
		t.Fatalf("creating Lina: %v", err)
	}
	nora, err := db.LoginOrCreateUser("Nora")
	if err != nil {
		t.Fatalf("creating Nora: %v", err)
	}

	sourceConversation, _, err := db.StartPrivateConversation(medi.ID, lina.ID)
	if err != nil {
		t.Fatalf("creating source conversation: %v", err)
	}
	destinationConversation, _, err := db.StartPrivateConversation(medi.ID, nora.ID)
	if err != nil {
		t.Fatalf("creating destination conversation: %v", err)
	}

	sourceText := "Forward and delete"
	source, err := db.SendMessage(medi.ID, sourceConversation.ID, "text", &sourceText, nil, nil)
	if err != nil {
		t.Fatalf("sending source message: %v", err)
	}
	replyText := "Reply that should lose reference"
	replyID := source.ID
	reply, err := db.SendMessage(lina.ID, sourceConversation.ID, "text", &replyText, nil, &replyID)
	if err != nil {
		t.Fatalf("sending reply: %v", err)
	}
	forwarded, err := db.ForwardMessage(medi.ID, source.ID, destinationConversation.ID)
	if err != nil {
		t.Fatalf("forwarding message: %v", err)
	}
	if _, err := db.CommentMessage(nora.ID, forwarded.ID, ":ok:"); err != nil {
		t.Fatalf("commenting forwarded message: %v", err)
	}
	if err := db.UncommentMessage(nora.ID, forwarded.ID); err != nil {
		t.Fatalf("uncommenting forwarded message: %v", err)
	}
	if err := db.DeleteMessage(medi.ID, source.ID); err != nil {
		t.Fatalf("deleting source message: %v", err)
	}

	reopened, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("reopening sqlite database: %v", err)
	}

	loadedSource, err := reopened.GetConversationForUser(lina.ID, sourceConversation.ID)
	if err != nil {
		t.Fatalf("loading source conversation: %v", err)
	}
	if len(loadedSource.Messages) != 1 || loadedSource.Messages[0].ID != reply.ID {
		t.Fatalf("expected only reply to remain, got %+v", loadedSource.Messages)
	}
	if loadedSource.Messages[0].ReplyToMessageID != nil {
		t.Fatalf("expected deleted reply reference to be cleared, got %+v", loadedSource.Messages[0])
	}

	loadedDestination, err := reopened.GetConversationForUser(nora.ID, destinationConversation.ID)
	if err != nil {
		t.Fatalf("loading destination conversation: %v", err)
	}
	if len(loadedDestination.Messages) != 1 || loadedDestination.Messages[0].ID != forwarded.ID {
		t.Fatalf("expected forwarded message, got %+v", loadedDestination.Messages)
	}
	if loadedDestination.Messages[0].Text == nil || *loadedDestination.Messages[0].Text != sourceText || len(loadedDestination.Messages[0].Reactions) != 0 {
		t.Fatalf("unexpected loaded forwarded message: %+v", loadedDestination.Messages[0])
	}
	if !loadedDestination.Messages[0].IsForwarded || loadedDestination.Messages[0].ForwardedFrom != "Medi" {
		t.Fatalf("expected forwarded metadata to persist, got %+v", loadedDestination.Messages[0])
	}
}
