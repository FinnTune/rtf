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

	defaultPostsPageSize = 10
	maxPostsPageSize     = 50

	maxSearchQueryLength = 100
	maxSearchResults     = 50

	defaultCommentsPageSize = 20
	maxCommentsPageSize     = 100
)

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

func validateComment(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxCommentLength {
		return "", fmt.Errorf("comment must be 1-%d characters", maxCommentLength)
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
