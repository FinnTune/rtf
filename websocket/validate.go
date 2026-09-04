package websocket

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)
	emailRegex    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

const (
	maxNameLength   = 50
	maxEmailLength  = 254
	maxGenderLength = 20
	minPasswordLen  = 8
	// bcrypt silently fails to hash inputs longer than 72 bytes; rejecting
	// them here instead of at HashPassword avoids storing a broken hash.
	maxPasswordLen = 72
	minAge         = 13
	maxAge         = 120

	maxPostTitleLength   = 100
	maxPostContentLength = 2000
	maxCommentLength     = 500
	maxCategoriesPerPost = 20
	maxCategoriesPerReq  = 50
	// category_name is VARCHAR(30) in the schema.
	maxCategoryNameLength = 30

	defaultPostsPageSize = 10
	maxPostsPageSize     = 50

	maxSearchQueryLength = 100
	maxSearchResults     = 50

	defaultCommentsPageSize = 20
	maxCommentsPageSize     = 100

	maxGroupNameLength = 50
	// Includes the creator, who's added automatically — a request naming
	// this many *other* usernames is still rejected, since the actual group
	// ends up one larger than what was requested.
	maxGroupMembers = 50

	// A chat message previously had no server-side length bound at all —
	// see sendMessage in ws-manager.go and the read-limit comment in
	// ws-client.go's readMessages.
	maxChatMessageLength = 1000
)

var validSortValues = map[string]bool{
	"newest":         true,
	"most_liked":     true,
	"most_commented": true,
}

// validateSortParam normalizes the sort query param, defaulting to "newest"
// when absent and rejecting anything outside the known set.
func validateSortParam(raw string) (string, error) {
	if raw == "" {
		return "newest", nil
	}
	if !validSortValues[raw] {
		return "", fmt.Errorf("invalid sort value: %s", raw)
	}
	return raw, nil
}

// validateRegistration trims string fields in place and rejects the request
// if any field is missing, malformed, or outside sane length bounds.
func validateRegistration(user *RegUser) error {
	user.Fname = strings.TrimSpace(user.Fname)
	user.Lname = strings.TrimSpace(user.Lname)
	user.Uname = strings.TrimSpace(user.Uname)
	user.Email = strings.TrimSpace(user.Email)
	user.Gender = strings.TrimSpace(user.Gender)

	if user.Fname == "" || len(user.Fname) > maxNameLength {
		return fmt.Errorf("first name must be 1-%d characters", maxNameLength)
	}
	if user.Lname == "" || len(user.Lname) > maxNameLength {
		return fmt.Errorf("last name must be 1-%d characters", maxNameLength)
	}
	if !usernameRegex.MatchString(user.Uname) {
		return fmt.Errorf("username must be 3-30 characters and contain only letters, numbers, underscores, or hyphens")
	}
	if user.Email == "" || len(user.Email) > maxEmailLength || !emailRegex.MatchString(user.Email) {
		return fmt.Errorf("a valid email address is required")
	}
	age, err := strconv.Atoi(user.Age)
	if err != nil || age < minAge || age > maxAge {
		return fmt.Errorf("age must be a number between %d and %d", minAge, maxAge)
	}
	if user.Gender == "" || len(user.Gender) > maxGenderLength {
		return fmt.Errorf("gender must be 1-%d characters", maxGenderLength)
	}
	if len(user.Pass) < minPasswordLen || len(user.Pass) > maxPasswordLen {
		return fmt.Errorf("password must be %d-%d characters", minPasswordLen, maxPasswordLen)
	}
	return nil
}

func validateLogin(username, password string) error {
	if strings.TrimSpace(username) == "" || len(username) > maxEmailLength {
		return fmt.Errorf("username is required")
	}
	if password == "" || len(password) > maxPasswordLen {
		return fmt.Errorf("password is required")
	}
	return nil
}

// validatePost trims title/content and checks them against length bounds,
// returning the trimmed values for the caller to store.
func validatePost(title, content string) (string, string, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || len(title) > maxPostTitleLength {
		return "", "", fmt.Errorf("title must be 1-%d characters", maxPostTitleLength)
	}
	if content == "" || len(content) > maxPostContentLength {
		return "", "", fmt.Errorf("content must be 1-%d characters", maxPostContentLength)
	}
	return title, content, nil
}

func validateCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxCategoryNameLength {
		return "", fmt.Errorf("category name must be 1-%d characters", maxCategoryNameLength)
	}
	return name, nil
}

func validateComment(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxCommentLength {
		return "", fmt.Errorf("comment must be 1-%d characters", maxCommentLength)
	}
	return content, nil
}

func validateChatMessage(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxChatMessageLength {
		return "", fmt.Errorf("message must be 1-%d characters", maxChatMessageLength)
	}
	return content, nil
}

func validateSearchQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" || len(q) > maxSearchQueryLength {
		return "", fmt.Errorf("search query must be 1-%d characters", maxSearchQueryLength)
	}
	return q, nil
}

// validateAuthorQuery checks an ?author= param against the same shape
// required at registration (usernameRegex), since post.author is always a
// real, previously-registered username.
func validateAuthorQuery(author string) (string, error) {
	author = strings.TrimSpace(author)
	if !usernameRegex.MatchString(author) {
		return "", fmt.Errorf("author must be 3-30 characters and contain only letters, numbers, underscores, or hyphens")
	}
	return author, nil
}

// escapeLikePattern escapes SQLite LIKE wildcard characters in user input so
// a search for a literal "%" or "_" doesn't get interpreted as a wildcard.
func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

// validateGroupChat trims the name and de-duplicates the member usernames
// for a "create-group-chat" request, rejecting an empty/oversized name or a
// member list that's empty or too large. Username existence itself isn't
// checked here — that requires a DB lookup, done by the caller once it
// knows the request shape itself is sane.
func validateGroupChat(name string, usernames []string) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxGroupNameLength {
		return "", nil, fmt.Errorf("group name must be 1-%d characters", maxGroupNameLength)
	}

	seen := make(map[string]bool, len(usernames))
	unique := make([]string, 0, len(usernames))
	for _, u := range usernames {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		unique = append(unique, u)
	}
	if len(unique) == 0 {
		return "", nil, fmt.Errorf("a group needs at least one other member")
	}
	if len(unique) > maxGroupMembers {
		return "", nil, fmt.Errorf("a group may have at most %d other members", maxGroupMembers)
	}

	return name, unique, nil
}
