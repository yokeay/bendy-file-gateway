package auth

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/bendy/file-gateway/internal/config"
	"github.com/bendy/file-gateway/internal/wasm"
)

// GitHubUser represents a GitHub user from the API.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubOAuthClient handles GitHub OAuth authentication.
type GitHubOAuthClient struct {
	clientID     string
	clientSecret string
	allowedUsers map[string]bool
}

// NewGitHubOAuthClient creates a new GitHub OAuth client from config.
func NewGitHubOAuthClient() *GitHubOAuthClient {
	allowedRaw := config.Get("ADMIN_GITHUB_USERNAMES")

	allowedUsers := map[string]bool{}
	for _, u := range strings.Split(allowedRaw, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			allowedUsers[u] = true
		}
	}

	return &GitHubOAuthClient{
		clientID:     config.Get("GITHUB_CLIENT_ID"),
		clientSecret: config.Get("GITHUB_CLIENT_SECRET"),
		allowedUsers: allowedUsers,
	}
}

// ExchangeCode exchanges an OAuth code for a GitHub access token.
func (c *GitHubOAuthClient) ExchangeCode(code string) (string, error) {
	resp, err := wasm.Fetch("POST", "https://github.com/login/oauth/access_token",
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "application/json",
		},
		fmt.Sprintf("client_id=%s&client_secret=%s&code=%s",
			url.QueryEscape(c.clientID),
			url.QueryEscape(c.clientSecret),
			url.QueryEscape(code)),
	)
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token exchange returned status %d: %s", resp.StatusCode, resp.Body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("token exchange error: %s", result.Error)
	}

	return result.AccessToken, nil
}

// GetUser fetches the GitHub user profile for an access token.
func (c *GitHubOAuthClient) GetUser(token string) (*GitHubUser, error) {
	resp, err := wasm.Fetch("GET", "https://api.github.com/user",
		map[string]string{
			"Authorization": "Bearer " + token,
			"Accept":        "application/vnd.github.v3+json",
		},
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Body)
	}

	var user GitHubUser
	if err := json.Unmarshal([]byte(resp.Body), &user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}
	return &user, nil
}

// IsAllowed checks if a GitHub username is in the allowed admin list.
func (c *GitHubOAuthClient) IsAllowed(username string) bool {
	return c.allowedUsers[username]
}

// GetAuthorizeURL returns the GitHub OAuth authorize URL.
func (c *GitHubOAuthClient) GetAuthorizeURL(redirectURI string) string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user",
		c.clientID,
		url.QueryEscape(redirectURI),
	)
}
