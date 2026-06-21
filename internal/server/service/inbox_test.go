package service

import (
	"strings"
	"testing"
)

func TestInboxListQueryThreadRepliesIncludeDirectMentionsOrRootCreator(t *testing.T) {
	if strings.Contains(listInboxQuery, "AND t.channel_id IN (") {
		t.Fatal("thread replies should not use channel-level participation")
	}
	parts := strings.Split(listInboxQuery, "-- DM messages")
	if len(parts) != 2 {
		t.Fatal("thread reply query section not found")
	}
	if strings.Contains(parts[0], "m.thread_id IN (") {
		t.Fatal("thread replies should not use reply participation")
	}
	if !strings.Contains(parts[0], "pm.sender_type = 'user' AND pm.sender_id = $1") {
		t.Fatal("thread replies should include replies to threads created by the user")
	}
	if !strings.Contains(parts[0], "WHERE um.message_id = m.id AND um.mentioned_user_id = $1") {
		t.Fatal("thread replies should require a direct mention on the reply")
	}
}

func TestInboxListQueryMentionsExcludeThreadReplies(t *testing.T) {
	parts := strings.Split(listInboxQuery, "-- @Mentions via user_mentions")
	if len(parts) != 2 {
		t.Fatal("mention query section not found")
	}
	if !strings.Contains(parts[1], "AND m.thread_id IS NULL") {
		t.Fatal("top-level mentions should exclude thread replies")
	}
}
