package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// NewGitHubOAuthClient creates a new GitHub OAuth client from environment variables.
func NewGitHubOAuthClient() *GitHubOAuthClient {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	allowedRaw := os.Getenv("ADMIN_GITHUB_USERNAMES")

	allowedUsers := map[string]bool{}
	for _, u := range strings.Split(allowedRaw, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			allowedUsers[u] = true
		}
	}

	return &GitHubOAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		allowedUsers: allowedUsers,
	}
}

// ExchangeCode exchanges an OAuth code for a GitHub access token.
func (c *GitHubOAuthClient) ExchangeCode(code string) (string, error) {
	resp, err := http.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
	})
	if err != nil {
		return "", fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	token := values.Get("access_token")
	if token == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return token, nil
}

// GetUser fetches the GitHub user profile for an access token.
func (c *GitHubOAuthClient) GetUser(token string) (*GitHubUser, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
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
