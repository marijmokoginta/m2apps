package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client interface {
	GetLatestRelease(owner, repo string) (*Release, error)
	GetReleaseByTag(owner, repo, tag string) (*Release, error)
	GetAllReleases(owner, repo string) ([]Release, error)
}

type APIClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewClient(token string) Client {
	return &APIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      strings.TrimSpace(token),
		baseURL:    "https://api.github.com",
	}
}

func (c *APIClient) GetReleaseByTag(owner, repo, tag string) (*Release, error) {
	escapedTag := url.PathEscape(strings.TrimSpace(tag))
	path := fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, escapedTag)
	return c.fetchRelease(path)
}

func (c *APIClient) GetLatestRelease(owner, repo string) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
	return c.fetchRelease(path)
}

func (c *APIClient) GetAllReleases(owner, repo string) ([]Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases", owner, repo)
	return c.fetchReleases(path)
}

func (c *APIClient) fetchRelease(path string) (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("github API unauthorized (401): invalid token")
	case http.StatusForbidden:
		return nil, fmt.Errorf("github API forbidden (403): access denied or rate limit exceeded")
	case http.StatusNotFound:
		return nil, fmt.Errorf("github release not found (404): check repository, tag, or access permissions")
	default:
		return nil, fmt.Errorf("github API request failed: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub release response: %w", err)
	}

	return &release, nil
}

func (c *APIClient) fetchReleases(path string) ([]Release, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("github API unauthorized (401): invalid token")
	case http.StatusForbidden:
		return nil, fmt.Errorf("github API forbidden (403): access denied or rate limit exceeded")
	case http.StatusNotFound:
		return nil, fmt.Errorf("github releases not found (404): check repository or access permissions")
	default:
		return nil, fmt.Errorf("github API request failed: %s", resp.Status)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub releases response: %w", err)
	}

	return releases, nil
}
