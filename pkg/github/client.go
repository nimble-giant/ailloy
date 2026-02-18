package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GraphQL query constants
const listProjectsQuery = `query($org: String!) {
  organization(login: $org) {
    projectsV2(first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        id
        number
        title
        url
        closed
      }
    }
  }
}`

const projectFieldsQuery = `query($org: String!, $number: Int!) {
  organization(login: $org) {
    projectV2(number: $number) {
      id
      title
      url
      fields(first: 50) {
        nodes {
          ... on ProjectV2Field {
            id
            name
            dataType
          }
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
          ... on ProjectV2IterationField {
            id
            name
            configuration {
              iterations {
                id
                title
              }
            }
          }
        }
      }
    }
  }
}`

// Execer abstracts command execution for testing
type Execer interface {
	Run(args []string) ([]byte, error)
}

// GHExecer calls the real gh CLI
type GHExecer struct{}

func (g *GHExecer) Run(args []string) ([]byte, error) {
	cmd := exec.Command("gh", args...) // #nosec G204 -- CLI tool invokes gh with controlled args
	return cmd.CombinedOutput()
}

// Client provides GitHub ProjectV2 discovery via gh api graphql
type Client struct {
	Exec  Execer
	cache map[string]any
}

// NewClient creates a new discovery client
func NewClient() *Client {
	return &Client{
		Exec:  &GHExecer{},
		cache: make(map[string]any),
	}
}

// CheckAuth validates that gh is installed and authenticated
func (c *Client) CheckAuth() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return ErrGHNotInstalled
	}

	out, err := c.Exec.Run([]string{"auth", "status"})
	if err != nil {
		if strings.Contains(string(out), "not logged") || strings.Contains(string(out), "not authenticated") {
			return ErrGHNotAuth
		}
		return fmt.Errorf("gh auth check failed: %s", string(out))
	}
	return nil
}

// ListProjects returns all ProjectV2 boards for an organization
func (c *Client) ListProjects(org string) ([]Project, error) {
	cacheKey := "projects:" + org
	if cached, ok := c.cache[cacheKey]; ok {
		return cached.([]Project), nil
	}

	out, err := c.Exec.Run([]string{
		"api", "graphql",
		"-f", "query=" + listProjectsQuery,
		"-f", "org=" + org,
	})
	if err != nil {
		return nil, c.parseError(out, err)
	}

	var resp graphQLResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if gqlErr := resp.toError(); gqlErr != nil {
		return nil, gqlErr
	}

	var data projectsData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse projects data: %w", err)
	}

	nodes := data.Organization.ProjectsV2.Nodes
	if len(nodes) == 0 {
		return nil, ErrNoProjects
	}

	projects := make([]Project, len(nodes))
	for i, n := range nodes {
		projects[i] = Project{
			ID:     n.ID,
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
			Closed: n.Closed,
		}
	}

	c.cache[cacheKey] = projects
	return projects, nil
}

// GetProjectFields returns all fields for a specific project
func (c *Client) GetProjectFields(org string, projectNumber int) (*DiscoveryResult, error) {
	cacheKey := fmt.Sprintf("fields:%s:%d", org, projectNumber)
	if cached, ok := c.cache[cacheKey]; ok {
		return cached.(*DiscoveryResult), nil
	}

	out, err := c.Exec.Run([]string{
		"api", "graphql",
		"-f", "query=" + projectFieldsQuery,
		"-f", "org=" + org,
		"-F", fmt.Sprintf("number=%d", projectNumber),
	})
	if err != nil {
		return nil, c.parseError(out, err)
	}

	var resp graphQLResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if gqlErr := resp.toError(); gqlErr != nil {
		return nil, gqlErr
	}

	var data projectFieldsData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse fields data: %w", err)
	}

	proj := data.Organization.ProjectV2
	result := &DiscoveryResult{
		Project: Project{
			ID:    proj.ID,
			Title: proj.Title,
			URL:   proj.URL,
		},
	}

	for _, raw := range proj.Fields.Nodes {
		field, err := parseFieldNode(raw)
		if err != nil {
			continue // skip unparseable fields
		}
		if field != nil {
			result.Fields = append(result.Fields, *field)
		}
	}

	c.cache[cacheKey] = result
	return result, nil
}

// parseError inspects gh output and exit error to return a specific sentinel error
func (c *Client) parseError(out []byte, execErr error) error {
	s := string(out)

	// Try to parse as GraphQL error response
	var resp graphQLResponse
	if json.Unmarshal(out, &resp) == nil {
		if gqlErr := resp.toError(); gqlErr != nil {
			return gqlErr
		}
	}

	if strings.Contains(s, "Could not resolve to an Organization") || strings.Contains(s, "NOT_FOUND") {
		return ErrOrgNotFound
	}
	if strings.Contains(s, "RATE_LIMITED") || strings.Contains(s, "rate limit") {
		return ErrRateLimited
	}
	if strings.Contains(s, "not logged") || strings.Contains(s, "not authenticated") {
		return ErrGHNotAuth
	}

	return fmt.Errorf("gh command failed: %s: %w", s, execErr)
}

// parseFieldNode unmarshals a polymorphic ProjectV2 field node
func parseFieldNode(raw json.RawMessage) (*Field, error) {
	// Probe for field type by checking for distinguishing keys
	var probe struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		DataType string `json:"dataType"`
		Options  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"options"`
		Configuration *struct {
			Iterations []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"iterations"`
		} `json:"configuration"`
	}

	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}

	// Skip empty nodes (can happen with inline fragments)
	if probe.ID == "" && probe.Name == "" {
		return nil, nil
	}

	field := &Field{
		ID:   probe.ID,
		Name: probe.Name,
	}

	switch {
	case len(probe.Options) > 0:
		field.Type = FieldTypeSingleSelect
		field.Options = make([]Option, len(probe.Options))
		for i, opt := range probe.Options {
			field.Options[i] = Option{ID: opt.ID, Name: opt.Name}
		}
	case probe.Configuration != nil:
		field.Type = FieldTypeIteration
		field.Options = make([]Option, len(probe.Configuration.Iterations))
		for i, iter := range probe.Configuration.Iterations {
			field.Options[i] = Option{ID: iter.ID, Name: iter.Title}
		}
	default:
		field.Type = mapDataType(probe.DataType)
	}

	return field, nil
}

func mapDataType(dt string) FieldType {
	switch dt {
	case "TEXT":
		return FieldTypeText
	case "NUMBER":
		return FieldTypeNumber
	case "DATE":
		return FieldTypeDate
	case "SINGLE_SELECT":
		return FieldTypeSingleSelect
	case "ITERATION":
		return FieldTypeIteration
	default:
		return FieldTypeUnknown
	}
}

// Internal response types for JSON unmarshalling

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (r *graphQLResponse) toError() error {
	if len(r.Errors) == 0 {
		return nil
	}
	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = e.Message
	}
	joined := strings.Join(msgs, "; ")

	if strings.Contains(joined, "Could not resolve to an Organization") {
		return ErrOrgNotFound
	}
	if strings.Contains(joined, "RATE_LIMITED") || strings.Contains(joined, "rate limit") {
		return ErrRateLimited
	}

	return &GraphQLError{Errors: msgs}
}

type projectsData struct {
	Organization struct {
		ProjectsV2 struct {
			Nodes []struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				URL    string `json:"url"`
				Closed bool   `json:"closed"`
			} `json:"nodes"`
		} `json:"projectsV2"`
	} `json:"organization"`
}

type projectFieldsData struct {
	Organization struct {
		ProjectV2 struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			URL    string `json:"url"`
			Fields struct {
				Nodes []json.RawMessage `json:"nodes"`
			} `json:"fields"`
		} `json:"projectV2"`
	} `json:"organization"`
}
