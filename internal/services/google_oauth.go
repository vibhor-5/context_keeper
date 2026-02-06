package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DevAnuragT/context_keeper/internal/config"
)

// GoogleOAuthService handles Google OAuth operations
type GoogleOAuthService interface {
	HandleGoogleCallback(ctx context.Context, code string) (*GoogleUserInfo, error)
	GetAuthURL(state string) string
}

// GoogleOAuthServiceImpl implements the GoogleOAuthService interface
type GoogleOAuthServiceImpl struct {
	config     *config.Config
	httpClient *http.Client
}

// GoogleUserInfo represents user information from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// GoogleTokenResponse represents the response from Google OAuth token exchange
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// NewGoogleOAuthService creates a new Google OAuth service
func NewGoogleOAuthService(cfg *config.Config) GoogleOAuthService {
	return &GoogleOAuthServiceImpl{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthURL returns the Google OAuth authorization URL
func (g *GoogleOAuthServiceImpl) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", g.config.GoogleOAuth.ClientID)
	params.Set("redirect_uri", g.config.GoogleOAuth.RedirectURL)
	params.Set("scope", "openid email profile")
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// HandleGoogleCallback handles the Google OAuth callback
func (g *GoogleOAuthServiceImpl) HandleGoogleCallback(ctx context.Context, code string) (*GoogleUserInfo, error) {
	// Exchange code for access token
	tokenResp, err := g.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information
	userInfo, err := g.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return userInfo, nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (g *GoogleOAuthServiceImpl) exchangeCodeForToken(ctx context.Context, code string) (*GoogleTokenResponse, error) {
	// Prepare request data
	data := url.Values{}
	data.Set("client_id", g.config.GoogleOAuth.ClientID)
	data.Set("client_secret", g.config.GoogleOAuth.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", g.config.GoogleOAuth.RedirectURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google OAuth token exchange failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("Google OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token received")
	}

	return &tokenResp, nil
}

// getUserInfo gets user information from Google API
func (g *GoogleOAuthServiceImpl) getUserInfo(ctx context.Context, token string) (*GoogleUserInfo, error) {
	// Get user profile
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API request failed with status: %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &userInfo, nil
}

// MockGoogleOAuthService is a mock implementation for testing
type MockGoogleOAuthService struct {
	HandleCallbackFunc func(ctx context.Context, code string) (*GoogleUserInfo, error)
	GetAuthURLFunc     func(state string) string
}

// NewMockGoogleOAuthService creates a new mock Google OAuth service
func NewMockGoogleOAuthService() *MockGoogleOAuthService {
	return &MockGoogleOAuthService{
		HandleCallbackFunc: func(ctx context.Context, code string) (*GoogleUserInfo, error) {
			return &GoogleUserInfo{
				ID:            "123456789",
				Email:         "test@example.com",
				VerifiedEmail: true,
				Name:          "Test User",
				GivenName:     "Test",
				FamilyName:    "User",
				Picture:       "https://example.com/avatar.jpg",
				Locale:        "en",
			}, nil
		},
		GetAuthURLFunc: func(state string) string {
			return "https://accounts.google.com/o/oauth2/v2/auth?mock=true&state=" + state
		},
	}
}

func (m *MockGoogleOAuthService) HandleGoogleCallback(ctx context.Context, code string) (*GoogleUserInfo, error) {
	return m.HandleCallbackFunc(ctx, code)
}

func (m *MockGoogleOAuthService) GetAuthURL(state string) string {
	return m.GetAuthURLFunc(state)
}