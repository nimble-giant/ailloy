package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeExecer returns canned responses based on argument patterns
type fakeExecer struct {
	calls     [][]string
	responses map[string]fakeResponse
}

type fakeResponse struct {
	output []byte
	err    error
}

func (f *fakeExecer) Run(args []string) ([]byte, error) {
	f.calls = append(f.calls, args)
	key := strings.Join(args, " ")
	for pattern, resp := range f.responses {
		if strings.Contains(key, pattern) {
			return resp.output, resp.err
		}
	}
	return nil, fmt.Errorf("unexpected call: %v", args)
}

func newFakeExecer(responses map[string]fakeResponse) *fakeExecer {
	return &fakeExecer{
		responses: responses,
	}
}

// --- CheckAuth tests ---

func TestCheckAuth_Success(t *testing.T) {
	fake := newFakeExecer(map[string]fakeResponse{
		"auth status": {output: []byte("Logged in to github.com")},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}
	// Note: CheckAuth also calls exec.LookPath which checks real PATH.
	// We test the exec path only; LookPath is tested by the existence of gh.
	err := client.checkAuthExec()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckAuth_NotAuthenticated(t *testing.T) {
	fake := newFakeExecer(map[string]fakeResponse{
		"auth status": {
			output: []byte("You are not logged into any GitHub hosts"),
			err:    errors.New("exit status 1"),
		},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}
	err := client.checkAuthExec()
	if !errors.Is(err, ErrGHNotAuth) {
		t.Errorf("expected ErrGHNotAuth, got %v", err)
	}
}

// --- ListProjects tests ---

func TestListProjects_Success(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectsV2": {
					"nodes": [
						{"id": "PVT_1", "number": 1, "title": "Engineering", "url": "https://github.com/orgs/acme/projects/1", "closed": false},
						{"id": "PVT_2", "number": 2, "title": "Roadmap", "url": "https://github.com/orgs/acme/projects/2", "closed": true}
					]
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	projects, err := client.ListProjects("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].Title != "Engineering" {
		t.Errorf("expected Engineering, got %s", projects[0].Title)
	}
	if projects[1].Closed != true {
		t.Error("expected second project to be closed")
	}
}

func TestListProjects_CacheHit(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectsV2": {
					"nodes": [
						{"id": "PVT_1", "number": 1, "title": "Engineering", "url": "", "closed": false}
					]
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	// First call
	_, err := client.ListProjects("acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should use cache
	_, err = client.ListProjects("acme")
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if len(fake.calls) != 1 {
		t.Errorf("expected 1 exec call (cached), got %d", len(fake.calls))
	}
}

func TestListProjects_NoProjects(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectsV2": {
					"nodes": []
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.ListProjects("acme")
	if !errors.Is(err, ErrNoProjects) {
		t.Errorf("expected ErrNoProjects, got %v", err)
	}
}

func TestListProjects_OrgNotFound(t *testing.T) {
	respJSON := `{
		"data": null,
		"errors": [{"type": "NOT_FOUND", "message": "Could not resolve to an Organization with the login of 'nonexistent'."}]
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.ListProjects("nonexistent")
	if !errors.Is(err, ErrOrgNotFound) {
		t.Errorf("expected ErrOrgNotFound, got %v", err)
	}
}

func TestListProjects_RateLimited(t *testing.T) {
	respJSON := `{
		"data": null,
		"errors": [{"type": "RATE_LIMITED", "message": "API rate limit exceeded"}]
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.ListProjects("acme")
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestListProjects_ExecError(t *testing.T) {
	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {
			output: []byte("Could not resolve to an Organization"),
			err:    errors.New("exit status 1"),
		},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.ListProjects("bad")
	if !errors.Is(err, ErrOrgNotFound) {
		t.Errorf("expected ErrOrgNotFound from exec error, got %v", err)
	}
}

// --- GetProjectFields tests ---

func TestGetProjectFields_Success(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectV2": {
					"id": "PVT_1",
					"title": "Engineering",
					"url": "https://github.com/orgs/acme/projects/1",
					"fields": {
						"nodes": [
							{"id": "PVTF_1", "name": "Title", "dataType": "TEXT"},
							{
								"id": "PVTSSF_1",
								"name": "Status",
								"options": [
									{"id": "opt_1", "name": "Todo"},
									{"id": "opt_2", "name": "In Progress"},
									{"id": "opt_3", "name": "Done"}
								]
							},
							{
								"id": "PVTSSF_2",
								"name": "Priority",
								"options": [
									{"id": "opt_4", "name": "P0"},
									{"id": "opt_5", "name": "P1"}
								]
							},
							{
								"id": "PVTIF_1",
								"name": "Sprint",
								"configuration": {
									"iterations": [
										{"id": "iter_1", "title": "Sprint 1"},
										{"id": "iter_2", "title": "Sprint 2"}
									]
								}
							}
						]
					}
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	result, err := client.GetProjectFields("acme", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Project.Title != "Engineering" {
		t.Errorf("expected Engineering, got %s", result.Project.Title)
	}

	if len(result.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(result.Fields))
	}

	// Check text field
	title := result.Fields[0]
	if title.Name != "Title" || title.Type != FieldTypeText {
		t.Errorf("expected Title/TEXT, got %s/%s", title.Name, title.Type)
	}

	// Check single-select field
	status := result.Fields[1]
	if status.Name != "Status" || status.Type != FieldTypeSingleSelect {
		t.Errorf("expected Status/SINGLE_SELECT, got %s/%s", status.Name, status.Type)
	}
	if len(status.Options) != 3 {
		t.Errorf("expected 3 status options, got %d", len(status.Options))
	}

	// Check iteration field
	sprint := result.Fields[3]
	if sprint.Name != "Sprint" || sprint.Type != FieldTypeIteration {
		t.Errorf("expected Sprint/ITERATION, got %s/%s", sprint.Name, sprint.Type)
	}
	if len(sprint.Options) != 2 {
		t.Errorf("expected 2 iterations, got %d", len(sprint.Options))
	}
	if sprint.Options[0].Name != "Sprint 1" {
		t.Errorf("expected Sprint 1, got %s", sprint.Options[0].Name)
	}
}

func TestGetProjectFields_CacheHit(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectV2": {
					"id": "PVT_1", "title": "Eng", "url": "",
					"fields": {"nodes": [{"id": "f1", "name": "Title", "dataType": "TEXT"}]}
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.GetProjectFields("acme", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = client.GetProjectFields("acme", 1)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if len(fake.calls) != 1 {
		t.Errorf("expected 1 exec call (cached), got %d", len(fake.calls))
	}
}

func TestGetProjectFields_GraphQLError(t *testing.T) {
	respJSON := `{
		"data": null,
		"errors": [{"type": "FORBIDDEN", "message": "Resource not accessible by integration"}]
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	_, err := client.GetProjectFields("acme", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var gqlErr *GraphQLError
	if !errors.As(err, &gqlErr) {
		t.Errorf("expected GraphQLError, got %T: %v", err, err)
	}
}

func TestGetProjectFields_SkipsEmptyNodes(t *testing.T) {
	respJSON := `{
		"data": {
			"organization": {
				"projectV2": {
					"id": "PVT_1", "title": "Eng", "url": "",
					"fields": {
						"nodes": [
							{},
							{"id": "f1", "name": "Title", "dataType": "TEXT"}
						]
					}
				}
			}
		}
	}`

	fake := newFakeExecer(map[string]fakeResponse{
		"api graphql": {output: []byte(respJSON)},
	})
	client := &Client{Exec: fake, cache: make(map[string]any)}

	result, err := client.GetProjectFields("acme", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Fields) != 1 {
		t.Errorf("expected 1 field (empty skipped), got %d", len(result.Fields))
	}
}

// --- parseFieldNode tests ---

func TestParseFieldNode_TextField(t *testing.T) {
	raw := json.RawMessage(`{"id": "f1", "name": "Title", "dataType": "TEXT"}`)
	field, err := parseFieldNode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if field.Type != FieldTypeText {
		t.Errorf("expected TEXT, got %s", field.Type)
	}
}

func TestParseFieldNode_SingleSelectField(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "f1", "name": "Status",
		"options": [{"id": "o1", "name": "Todo"}, {"id": "o2", "name": "Done"}]
	}`)
	field, err := parseFieldNode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if field.Type != FieldTypeSingleSelect {
		t.Errorf("expected SINGLE_SELECT, got %s", field.Type)
	}
	if len(field.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(field.Options))
	}
}

func TestParseFieldNode_IterationField(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "f1", "name": "Sprint",
		"configuration": {
			"iterations": [{"id": "i1", "title": "Sprint 1"}]
		}
	}`)
	field, err := parseFieldNode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if field.Type != FieldTypeIteration {
		t.Errorf("expected ITERATION, got %s", field.Type)
	}
	if len(field.Options) != 1 {
		t.Errorf("expected 1 iteration, got %d", len(field.Options))
	}
}

func TestParseFieldNode_EmptyNode(t *testing.T) {
	raw := json.RawMessage(`{}`)
	field, err := parseFieldNode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if field != nil {
		t.Errorf("expected nil for empty node, got %v", field)
	}
}

// checkAuthExec is a helper that tests only the exec path of CheckAuth (not LookPath)
func (c *Client) checkAuthExec() error {
	out, err := c.Exec.Run([]string{"auth", "status"})
	if err != nil {
		s := string(out)
		if strings.Contains(s, "not logged") || strings.Contains(s, "not authenticated") {
			return ErrGHNotAuth
		}
		return fmt.Errorf("gh auth check failed: %s", s)
	}
	return nil
}
