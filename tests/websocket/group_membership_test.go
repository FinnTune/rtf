package websocket_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

// mustCreateGroupChat drives the create-group-chat request/response round
// trip for the creator and returns the resulting conversation info, failing
// the test on any error or unexpected event. Mirrors
// TestCreateGroupChat_CreatesConversationAndNotifiesOnlineMembers's own
// inline version, factored out since every test in this file needs one.
func mustCreateGroupChat(t *testing.T, creator *websocket.TestClientHandle, name string, usernames []string) websocket.ConversationInfo {
	t.Helper()
	payload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: name, Usernames: usernames})
	if err := websocket.CreateGroupChatForTest(payload, creator); err != nil {
		t.Fatalf("createGroupChat failed: %v", err)
	}
	eventType, eventPayload, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-opened")
	}
	if eventType != websocket.ChatOpened {
		t.Fatalf("expected chat-opened event, got %q", eventType)
	}
	var info websocket.ConversationInfo
	if err := json.Unmarshal(eventPayload, &info); err != nil {
		t.Fatalf("failed to decode chat-opened: %v", err)
	}
	return info
}

func hasMember(members []websocket.ConversationMember, username string) bool {
	for _, m := range members {
		if m.Username == username {
			return true
		}
	}
	return false
}

func TestLeaveGroup_RemovesMemberAndNotifiesEveryone(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	alice := websocket.AddTestClient("s2", "alice", 2)
	actualUser := websocket.AddTestClient("s3", "actual_user", 42)

	info := mustCreateGroupChat(t, creator, "Trip Planning", []string{"alice", "actual_user"})
	// Drain the chat-opened pushes createGroupChat sent to the two other
	// online members, so WaitEvent below sees only the leave notification.
	alice.WaitEvent(time.Second)
	actualUser.WaitEvent(time.Second)

	payload, _ := json.Marshal(websocket.LeaveGroupRequest{ConversationID: info.ConversationID})
	if err := websocket.LeaveGroupForTest(payload, alice); err != nil {
		t.Fatalf("leaveGroup failed: %v", err)
	}

	for _, recipient := range []*websocket.TestClientHandle{creator, alice, actualUser} {
		eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
		if !ok {
			t.Fatalf("timed out waiting for group-membership-changed (%s)", recipient.Username())
		}
		if eventType != websocket.GroupMembershipChanged {
			t.Fatalf("expected group-membership-changed for %s, got %q", recipient.Username(), eventType)
		}
		var updated websocket.ConversationInfo
		if err := json.Unmarshal(eventPayload, &updated); err != nil {
			t.Fatalf("failed to decode group-membership-changed: %v", err)
		}
		if hasMember(updated.Members, "alice") {
			t.Fatalf("expected alice to be gone from the member list seen by %s, got %+v", recipient.Username(), updated.Members)
		}
		if len(updated.Members) != 2 {
			t.Fatalf("expected 2 remaining members in the payload seen by %s, got %d", recipient.Username(), len(updated.Members))
		}
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 2 {
		t.Fatalf("expected 2 persisted members after leaving, got %d", memberCount)
	}
}

func TestLeaveGroup_RejectsNonMember(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	outsider := websocket.AddTestClient("s2", "alice", 2)

	info := mustCreateGroupChat(t, creator, "Solo Group", []string{"actual_user"})

	payload, _ := json.Marshal(websocket.LeaveGroupRequest{ConversationID: info.ConversationID})
	if err := websocket.LeaveGroupForTest(payload, outsider); err != nil {
		t.Fatalf("leaveGroup should not error, got: %v", err)
	}

	eventType, _, ok := outsider.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestLeaveGroup_RejectsDirectConversation(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	admin := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, admin, "actual_user")

	payload, _ := json.Marshal(websocket.LeaveGroupRequest{ConversationID: info.ConversationID})
	if err := websocket.LeaveGroupForTest(payload, admin); err != nil {
		t.Fatalf("leaveGroup should not error, got: %v", err)
	}

	eventType, _, ok := admin.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

// TestLeaveGroup_DeletesConversationWhenLastMemberLeaves guards the cleanup
// half of removeConversationMember: nothing else in this schema ever
// deletes a conversation, so a group that loses every member must not sit
// in the database forever as an unreachable, permanently-orphaned row.
func TestLeaveGroup_DeletesConversationWhenLastMemberLeaves(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	other := websocket.AddTestClient("s2", "alice", 2)

	info := mustCreateGroupChat(t, creator, "Shrinking Group", []string{"alice"})
	other.WaitEvent(time.Second) // drain alice's own chat-opened push

	leavePayload, _ := json.Marshal(websocket.LeaveGroupRequest{ConversationID: info.ConversationID})
	if err := websocket.LeaveGroupForTest(leavePayload, other); err != nil {
		t.Fatalf("first leaveGroup failed: %v", err)
	}
	creator.WaitEvent(time.Second)
	other.WaitEvent(time.Second)

	if err := websocket.LeaveGroupForTest(leavePayload, creator); err != nil {
		t.Fatalf("second leaveGroup failed: %v", err)
	}
	// The last leaver still gets a group-membership-changed push (with an
	// empty member list) even though the conversation row is now gone.
	eventType, eventPayload, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for the final group-membership-changed")
	}
	if eventType != websocket.GroupMembershipChanged {
		t.Fatalf("expected group-membership-changed, got %q", eventType)
	}
	var final websocket.ConversationInfo
	if err := json.Unmarshal(eventPayload, &final); err != nil {
		t.Fatalf("failed to decode final group-membership-changed: %v", err)
	}
	if len(final.Members) != 0 {
		t.Fatalf("expected an empty member list once the conversation is gone, got %+v", final.Members)
	}

	var convCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation WHERE id = ?`, info.ConversationID).Scan(&convCount); err != nil {
		t.Fatalf("failed to count conversation rows: %v", err)
	}
	if convCount != 0 {
		t.Fatalf("expected the emptied conversation to be deleted, but it still exists")
	}
}

func TestAddGroupMember_AddsNewMemberAndNotifiesEveryone(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	alice := websocket.AddTestClient("s2", "alice", 2)
	newMember := websocket.AddTestClient("s3", "actual_user", 42)

	info := mustCreateGroupChat(t, creator, "Growing Group", []string{"alice"})
	alice.WaitEvent(time.Second) // drain alice's own chat-opened push

	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: "actual_user"})
	if err := websocket.AddGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("addGroupMember failed: %v", err)
	}

	for _, recipient := range []*websocket.TestClientHandle{creator, alice, newMember} {
		eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
		if !ok {
			t.Fatalf("timed out waiting for group-membership-changed (%s)", recipient.Username())
		}
		if eventType != websocket.GroupMembershipChanged {
			t.Fatalf("expected group-membership-changed for %s, got %q", recipient.Username(), eventType)
		}
		var updated websocket.ConversationInfo
		if err := json.Unmarshal(eventPayload, &updated); err != nil {
			t.Fatalf("failed to decode group-membership-changed: %v", err)
		}
		if !hasMember(updated.Members, "actual_user") {
			t.Fatalf("expected actual_user in the member list seen by %s, got %+v", recipient.Username(), updated.Members)
		}
		if len(updated.Members) != 3 {
			t.Fatalf("expected 3 members in the payload seen by %s, got %d", recipient.Username(), len(updated.Members))
		}
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 3 {
		t.Fatalf("expected 3 persisted members after adding, got %d", memberCount)
	}
}

func TestAddGroupMember_RejectsAlreadyMember(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	alice := websocket.AddTestClient("s2", "alice", 2)
	info := mustCreateGroupChat(t, creator, "Full Group", []string{"alice"})
	alice.WaitEvent(time.Second)

	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: "alice"})
	if err := websocket.AddGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("addGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestAddGroupMember_RejectsUnknownUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	info := mustCreateGroupChat(t, creator, "Group", []string{"alice"})

	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: "does-not-exist"})
	if err := websocket.AddGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("addGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

// seedSyntheticUsers inserts n minimal, distinct user rows directly (not
// through registration, which isn't what these size-cap tests are about)
// and returns their usernames — cheap bulk setup for tests that need many
// real users to fill out a group's member list.
func seedSyntheticUsers(t *testing.T, db *sql.DB, n int, prefix string) []string {
	t.Helper()
	usernames := make([]string, n)
	for i := 0; i < n; i++ {
		uname := fmt.Sprintf("%s%d", prefix, i)
		usernames[i] = uname
		if _, err := db.Exec(
			`INSERT INTO user (fname, lname, uname, email, age, gender, pass, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
			"Synthetic", "User", uname, uname+"@example.com", "30", "other", "hash",
		); err != nil {
			t.Fatalf("failed to seed synthetic user %q: %v", uname, err)
		}
	}
	return usernames
}

// addSyntheticConversationMembers adds every named (already-seeded) user to
// convID directly via SQL, bypassing the add-group-member event entirely —
// used to cheaply grow a group to a specific size for the size-cap tests
// below, without needing that many real, connected WS clients.
func addSyntheticConversationMembers(t *testing.T, db *sql.DB, convID int, usernames []string) {
	t.Helper()
	for _, uname := range usernames {
		var userID int
		if err := db.QueryRow(`SELECT id FROM user WHERE uname = ?`, uname).Scan(&userID); err != nil {
			t.Fatalf("failed to look up seeded user %q: %v", uname, err)
		}
		if _, err := db.Exec(
			`INSERT INTO conversation_member (conversation_id, user_id, joined_at) VALUES (?, ?, datetime('now'))`,
			convID, userID,
		); err != nil {
			t.Fatalf("failed to add %q to conversation %d: %v", uname, convID, err)
		}
	}
}

// TestAddGroupMember_RejectsWhenGroupIsAtMaxSize guards the actual fix:
// addGroupMember must enforce the same size cap createGroupChat already
// does via validateGroupChat (maxGroupMembers other members, plus the
// creator) — without this, any member could grow a group past that limit
// one add-group-member call at a time.
func TestAddGroupMember_RejectsWhenGroupIsAtMaxSize(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	others := seedSyntheticUsers(t, db, websocket.MaxGroupMembersForTest, "capmember")
	info := mustCreateGroupChat(t, creator, "Big Group", []string{others[0]})
	// createGroupChat already added others[0] — fill in the rest to reach
	// exactly maxGroupMembers other members (maxGroupMembers+1 total, the
	// same ceiling validateGroupChat enforces at creation time).
	addSyntheticConversationMembers(t, db, info.ConversationID, others[1:])

	oneMore := seedSyntheticUsers(t, db, 1, "onemore")[0]
	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: oneMore})
	if err := websocket.AddGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("addGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != websocket.MaxGroupMembersForTest+1 {
		t.Fatalf("expected the rejected add to leave the member count unchanged at %d, got %d", websocket.MaxGroupMembersForTest+1, memberCount)
	}
}

// TestAddGroupMember_AllowsAddingUpToMaxSize is the boundary-condition
// sibling of the rejection test above: a group one below the cap must
// still accept exactly one more member, ending at the same ceiling
// createGroupChat's own validateGroupChat allows.
func TestAddGroupMember_AllowsAddingUpToMaxSize(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	others := seedSyntheticUsers(t, db, websocket.MaxGroupMembersForTest-1, "capmember")
	info := mustCreateGroupChat(t, creator, "Almost Full Group", []string{others[0]})
	addSyntheticConversationMembers(t, db, info.ConversationID, others[1:])

	var memberCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCountBefore); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCountBefore != websocket.MaxGroupMembersForTest {
		t.Fatalf("test setup bug: expected %d members before the final add, got %d", websocket.MaxGroupMembersForTest, memberCountBefore)
	}

	lastSlot := seedSyntheticUsers(t, db, 1, "lastslot")[0]
	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: lastSlot})
	if err := websocket.AddGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("addGroupMember failed: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for group-membership-changed")
	}
	if eventType != websocket.GroupMembershipChanged {
		t.Fatalf("expected group-membership-changed event, got %q", eventType)
	}

	var memberCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCountAfter); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCountAfter != websocket.MaxGroupMembersForTest+1 {
		t.Fatalf("expected the successful add to bring the member count to %d, got %d", websocket.MaxGroupMembersForTest+1, memberCountAfter)
	}
}

func TestAddGroupMember_RejectsNonMemberRequester(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	outsider := websocket.AddTestClient("s2", "alice", 2)
	info := mustCreateGroupChat(t, creator, "Group", []string{"actual_user"})

	payload, _ := json.Marshal(websocket.AddGroupMemberRequest{ConversationID: info.ConversationID, Username: "alice"})
	if err := websocket.AddGroupMemberForTest(payload, outsider); err != nil {
		t.Fatalf("addGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := outsider.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestRemoveGroupMember_RemovesTargetAndNotifiesEveryone(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	alice := websocket.AddTestClient("s2", "alice", 2)
	actualUser := websocket.AddTestClient("s3", "actual_user", 42)

	info := mustCreateGroupChat(t, creator, "Group", []string{"alice", "actual_user"})
	alice.WaitEvent(time.Second)
	actualUser.WaitEvent(time.Second)

	payload, _ := json.Marshal(websocket.RemoveGroupMemberRequest{ConversationID: info.ConversationID, Username: "alice"})
	if err := websocket.RemoveGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("removeGroupMember failed: %v", err)
	}

	for _, recipient := range []*websocket.TestClientHandle{creator, alice, actualUser} {
		eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
		if !ok {
			t.Fatalf("timed out waiting for group-membership-changed (%s)", recipient.Username())
		}
		if eventType != websocket.GroupMembershipChanged {
			t.Fatalf("expected group-membership-changed for %s, got %q", recipient.Username(), eventType)
		}
		var updated websocket.ConversationInfo
		if err := json.Unmarshal(eventPayload, &updated); err != nil {
			t.Fatalf("failed to decode group-membership-changed: %v", err)
		}
		if hasMember(updated.Members, "alice") {
			t.Fatalf("expected alice to be gone from the member list seen by %s, got %+v", recipient.Username(), updated.Members)
		}
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 2 {
		t.Fatalf("expected 2 persisted members after removal, got %d", memberCount)
	}
}

func TestRemoveGroupMember_RejectsNonMemberTarget(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	info := mustCreateGroupChat(t, creator, "Group", []string{"actual_user"})

	payload, _ := json.Marshal(websocket.RemoveGroupMemberRequest{ConversationID: info.ConversationID, Username: "alice"})
	if err := websocket.RemoveGroupMemberForTest(payload, creator); err != nil {
		t.Fatalf("removeGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestRemoveGroupMember_RejectsNonMemberRequester(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	outsider := websocket.AddTestClient("s2", "alice", 2)
	info := mustCreateGroupChat(t, creator, "Group", []string{"actual_user"})

	payload, _ := json.Marshal(websocket.RemoveGroupMemberRequest{ConversationID: info.ConversationID, Username: "actual_user"})
	if err := websocket.RemoveGroupMemberForTest(payload, outsider); err != nil {
		t.Fatalf("removeGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := outsider.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestRemoveGroupMember_RejectsDirectConversation(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	admin := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, admin, "actual_user")

	payload, _ := json.Marshal(websocket.RemoveGroupMemberRequest{ConversationID: info.ConversationID, Username: "actual_user"})
	if err := websocket.RemoveGroupMemberForTest(payload, admin); err != nil {
		t.Fatalf("removeGroupMember should not error, got: %v", err)
	}

	eventType, _, ok := admin.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}
