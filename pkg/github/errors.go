package github

import (
	"errors"
	"strings"
)

var (
	ErrGHNotInstalled  = errors.New("gh CLI is not installed or not on PATH")
	ErrGHNotAuth       = errors.New("gh CLI is not authenticated; run 'gh auth login'")
	ErrOrgNotFound     = errors.New("organization not found or not accessible")
	ErrNoProjects      = errors.New("no ProjectV2 boards found for this organization")
	ErrProjectNotFound = errors.New("specified project not found")
	ErrRateLimited     = errors.New("GitHub API rate limit exceeded; try again later")
)

// GraphQLError wraps one or more errors returned by the GitHub GraphQL API
type GraphQLError struct {
	Errors []string
}

func (e *GraphQLError) Error() string {
	return "graphql error: " + strings.Join(e.Errors, "; ")
}
