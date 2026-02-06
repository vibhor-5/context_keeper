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
	"github.com/DevAnuragT/context_keeper/internal/models"
)

// AuthServiceImpl implements the AuthService interface
type AuthServiceImpl struct {
	config         *config.Config
	httpClient     *http.Client
	store          RepositoryStore
	passwordSvc    PasswordService
	emailSvc       EmailService
	googleOAuthSvc GoogleOAuthService
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.Config, store RepositoryStore, passwordSvc PasswordService, emailSvc EmailService, googleOAuthSvc GoogleOAuthService) AuthService {
	return &AuthServiceImpl{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store:          store,
		passwordSvc:    passwordSvc,
		emailSvc:       emailSvc,
		googleOAuthSvc: googleOAuthSvc,
	}
}

// RegisterUser registers a new user with email and password
func (a *AuthServiceImpl) RegisterUser(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	existingUser, err := a.store.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash password
	passwordHash, err := a.passwordSvc.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate email verification token
	verificationToken, err := a.passwordSvc.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Create user
	user := &models.User{
		Email:                      req.Email,
		PasswordHash:               &passwordHash,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		EmailVerified:              false,
		EmailVerificationToken:     &verificationToken,
		EmailVerificationExpiresAt: authTimePtr(time.Now().Add(24 * time.Hour)),
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	if err := a.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Send verification email
	firstName := ""
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	if err := a.emailSvc.SendEmailVerification(ctx, req.Email, verificationToken, firstName); err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &models.AuthResponse{
		Token: jwtToken,
		User:  *user,
	}, nil
}

// LoginWithEmailPassword authenticates a user with email and password
func (a *AuthServiceImpl) LoginWithEmailPassword(ctx context.Context, req *models.EmailPasswordAuthRequest) (*models.AuthResponse, error) {
	// Get user by email
	user, err := a.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if user has a password (not OAuth-only)
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("this account uses OAuth login. Please use GitHub or Google to sign in")
	}

	// Verify password
	valid, err := a.passwordSvc.VerifyPassword(req.Password, *user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	if !valid {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err := a.store.UpdateUser(ctx, user.ID, map[string]interface{}{
		"last_login_at": now,
	}); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login time: %v\n", err)
	}

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &models.AuthResponse{
		Token: jwtToken,
		User:  *user,
	}, nil
}

// HandleOAuthCallback handles OAuth callbacks for multiple providers
func (a *AuthServiceImpl) HandleOAuthCallback(ctx context.Context, provider string, code string) (*models.AuthResponse, error) {
	switch provider {
	case "github":
		return a.HandleGitHubCallback(ctx, code)
	case "google":
		return a.handleGoogleCallback(ctx, code)
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// handleGoogleCallback handles Google OAuth callback
func (a *AuthServiceImpl) handleGoogleCallback(ctx context.Context, code string) (*models.AuthResponse, error) {
	// Get user info from Google
	googleUser, err := a.googleOAuthSvc.HandleGoogleCallback(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to handle Google callback: %w", err)
	}

	// Check if OAuth account already exists
	existingOAuth, err := a.store.GetOAuthAccount(ctx, "google", googleUser.ID)
	if err == nil && existingOAuth != nil {
		// User exists, get the user and update last login
		user, err := a.store.GetUserByID(ctx, existingOAuth.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Update last login time
		now := time.Now()
		user.LastLoginAt = &now
		if err := a.store.UpdateUser(ctx, user.ID, map[string]interface{}{
			"last_login_at": now,
		}); err != nil {
			fmt.Printf("Failed to update last login time: %v\n", err)
		}

		// Populate legacy fields for backward compatibility
		user.Login = googleUser.Email
		// Note: Google doesn't provide GitHub token, so GitHubToken remains empty

		// Generate JWT token
		jwtToken, err := a.GenerateJWT(user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT: %w", err)
		}

		return &models.AuthResponse{
			Token: jwtToken,
			User:  *user,
		}, nil
	}

	// Check if user exists with this email
	existingUser, err := a.store.GetUserByEmail(ctx, googleUser.Email)
	if err == nil && existingUser != nil {
		// Link Google account to existing user
		oauthAccount := &models.UserOAuthAccount{
			UserID:           existingUser.ID,
			Provider:         "google",
			ProviderUserID:   googleUser.ID,
			ProviderUsername: &googleUser.Email,
			AccessToken:      "google_token", // We don't store the actual token for security
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := a.store.CreateOAuthAccount(ctx, oauthAccount); err != nil {
			return nil, fmt.Errorf("failed to link Google account: %w", err)
		}

		// Update user info from Google if not set
		updates := make(map[string]interface{})
		if existingUser.FirstName == nil && googleUser.GivenName != "" {
			updates["first_name"] = googleUser.GivenName
		}
		if existingUser.LastName == nil && googleUser.FamilyName != "" {
			updates["last_name"] = googleUser.FamilyName
		}
		if existingUser.AvatarURL == nil && googleUser.Picture != "" {
			updates["avatar_url"] = googleUser.Picture
		}
		if googleUser.VerifiedEmail && !existingUser.EmailVerified {
			updates["email_verified"] = true
		}
		updates["last_login_at"] = time.Now()

		if len(updates) > 0 {
			if err := a.store.UpdateUser(ctx, existingUser.ID, updates); err != nil {
				fmt.Printf("Failed to update user info: %v\n", err)
			}
		}

		// Populate legacy fields for backward compatibility
		existingUser.Login = googleUser.Email

		// Generate JWT token
		jwtToken, err := a.GenerateJWT(existingUser)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT: %w", err)
		}

		return &models.AuthResponse{
			Token: jwtToken,
			User:  *existingUser,
		}, nil
	}

	// Create new user
	user := &models.User{
		Email:         googleUser.Email,
		FirstName:     authStringPtr(googleUser.GivenName),
		LastName:      authStringPtr(googleUser.FamilyName),
		AvatarURL:     authStringPtr(googleUser.Picture),
		EmailVerified: googleUser.VerifiedEmail,
		LastLoginAt:   authTimePtr(time.Now()),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Login:         googleUser.Email, // For backward compatibility
	}

	if err := a.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create OAuth account
	oauthAccount := &models.UserOAuthAccount{
		UserID:           user.ID,
		Provider:         "google",
		ProviderUserID:   googleUser.ID,
		ProviderUsername: &googleUser.Email,
		AccessToken:      "google_token", // We don't store the actual token for security
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := a.store.CreateOAuthAccount(ctx, oauthAccount); err != nil {
		return nil, fmt.Errorf("failed to create OAuth account: %w", err)
	}

	// Send welcome email
	firstName := ""
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if err := a.emailSvc.SendWelcomeEmail(ctx, user.Email, firstName); err != nil {
		fmt.Printf("Failed to send welcome email: %v\n", err)
	}

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &models.AuthResponse{
		Token: jwtToken,
		User:  *user,
	}, nil
}

// RequestPasswordReset initiates a password reset flow
func (a *AuthServiceImpl) RequestPasswordReset(ctx context.Context, email string) error {
	// Get user by email
	user, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return nil
	}

	// Check if user has a password (not OAuth-only)
	if user.PasswordHash == nil {
		// Don't reveal if user is OAuth-only for security
		return nil
	}

	// Generate reset token
	resetToken, err := a.passwordSvc.GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Update user with reset token
	if err := a.store.UpdateUser(ctx, user.ID, map[string]interface{}{
		"password_reset_token":      resetToken,
		"password_reset_expires_at": time.Now().Add(1 * time.Hour),
	}); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	// Send reset email
	firstName := ""
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if err := a.emailSvc.SendPasswordReset(ctx, email, resetToken, firstName); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	return nil
}

// ResetPassword completes the password reset flow
func (a *AuthServiceImpl) ResetPassword(ctx context.Context, token, newPassword string) error {
	// This is a simplified implementation - in production, you'd want to:
	// 1. Find user by reset token
	// 2. Check token expiration
	// 3. Update password and clear token
	
	return fmt.Errorf("password reset not fully implemented - requires additional store methods")
}

// VerifyEmail verifies a user's email address
func (a *AuthServiceImpl) VerifyEmail(ctx context.Context, token string) error {
	// Find user by verification token and verify
	// This is a simplified implementation - requires additional store methods
	return fmt.Errorf("email verification not fully implemented - requires additional store methods")
}

// ResendEmailVerification resends email verification
func (a *AuthServiceImpl) ResendEmailVerification(ctx context.Context, email string) error {
	user, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Generate new verification token
	verificationToken, err := a.passwordSvc.GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Update user with new token
	if err := a.store.UpdateUser(ctx, user.ID, map[string]interface{}{
		"email_verification_token":      verificationToken,
		"email_verification_expires_at": time.Now().Add(24 * time.Hour),
	}); err != nil {
		return fmt.Errorf("failed to save verification token: %w", err)
	}

	// Send verification email
	firstName := ""
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if err := a.emailSvc.SendEmailVerification(ctx, email, verificationToken, firstName); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// GetUserByID gets a user by ID
func (a *AuthServiceImpl) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return a.store.GetUserByID(ctx, userID)
}

// GetUserByEmail gets a user by email
func (a *AuthServiceImpl) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return a.store.GetUserByEmail(ctx, email)
}

// UpdateUser updates user information
func (a *AuthServiceImpl) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	return a.store.UpdateUser(ctx, userID, updates)
}

// LinkOAuthAccount links an OAuth account to a user
func (a *AuthServiceImpl) LinkOAuthAccount(ctx context.Context, userID string, provider string, account *models.UserOAuthAccount) error {
	account.UserID = userID
	return a.store.CreateOAuthAccount(ctx, account)
}

// UnlinkOAuthAccount unlinks an OAuth account from a user
func (a *AuthServiceImpl) UnlinkOAuthAccount(ctx context.Context, userID string, provider string) error {
	// This requires additional store methods to find and delete by user ID and provider
	return fmt.Errorf("unlink OAuth account not fully implemented - requires additional store methods")
}

// GetOAuthAccounts gets all OAuth accounts for a user
func (a *AuthServiceImpl) GetOAuthAccounts(ctx context.Context, userID string) ([]models.UserOAuthAccount, error) {
	return a.store.GetOAuthAccountsByUser(ctx, userID)
}

// HandleGitHubCallback handles the GitHub OAuth callback (legacy method for backward compatibility)
func (a *AuthServiceImpl) HandleGitHubCallback(ctx context.Context, code string) (*models.AuthResponse, error) {
	// Exchange code for access token
	token, err := a.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information from GitHub
	githubUser, err := a.getGitHubUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if OAuth account already exists
	existingOAuth, err := a.store.GetOAuthAccount(ctx, "github", githubUser.ID)
	if err == nil && existingOAuth != nil {
		// User exists, get the user and update last login
		user, err := a.store.GetUserByID(ctx, existingOAuth.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Update OAuth account token
		if err := a.store.UpdateOAuthAccount(ctx, existingOAuth.ID, map[string]interface{}{
			"access_token": token,
			"updated_at":   time.Now(),
		}); err != nil {
			fmt.Printf("Failed to update OAuth token: %v\n", err)
		}

		// Update last login time
		now := time.Now()
		user.LastLoginAt = &now
		if err := a.store.UpdateUser(ctx, user.ID, map[string]interface{}{
			"last_login_at": now,
		}); err != nil {
			fmt.Printf("Failed to update last login time: %v\n", err)
		}

		// Populate legacy fields for backward compatibility
		user.Login = githubUser.Login
		user.GitHubToken = token

		// Generate JWT token
		jwtToken, err := a.GenerateJWT(user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT: %w", err)
		}

		return &models.AuthResponse{
			Token: jwtToken,
			User:  *user,
		}, nil
	}

	// Check if user exists with this email
	email := githubUser.Email
	if email == "" {
		// Try to get email from GitHub API
		email, _ = a.getUserEmail(ctx, token)
	}

	var user *models.User
	if email != "" {
		existingUser, err := a.store.GetUserByEmail(ctx, email)
		if err == nil && existingUser != nil {
			user = existingUser
		}
	}

	if user != nil {
		// Link GitHub account to existing user
		oauthAccount := &models.UserOAuthAccount{
			UserID:           user.ID,
			Provider:         "github",
			ProviderUserID:   githubUser.ID,
			ProviderUsername: &githubUser.Login,
			AccessToken:      token,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := a.store.CreateOAuthAccount(ctx, oauthAccount); err != nil {
			return nil, fmt.Errorf("failed to link GitHub account: %w", err)
		}

		// Update user info from GitHub if not set
		updates := make(map[string]interface{})
		if user.AvatarURL == nil && githubUser.AvatarURL != "" {
			updates["avatar_url"] = githubUser.AvatarURL
		}
		updates["last_login_at"] = time.Now()

		if len(updates) > 0 {
			if err := a.store.UpdateUser(ctx, user.ID, updates); err != nil {
				fmt.Printf("Failed to update user info: %v\n", err)
			}
		}
	} else {
		// Create new user
		user = &models.User{
			Email:         email,
			EmailVerified: email != "", // GitHub emails are considered verified
			LastLoginAt:   authTimePtr(time.Now()),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := a.store.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Create OAuth account
		oauthAccount := &models.UserOAuthAccount{
			UserID:           user.ID,
			Provider:         "github",
			ProviderUserID:   githubUser.ID,
			ProviderUsername: &githubUser.Login,
			AccessToken:      token,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := a.store.CreateOAuthAccount(ctx, oauthAccount); err != nil {
			return nil, fmt.Errorf("failed to create OAuth account: %w", err)
		}

		// Send welcome email if we have an email
		if email != "" {
			if err := a.emailSvc.SendWelcomeEmail(ctx, email, ""); err != nil {
				fmt.Printf("Failed to send welcome email: %v\n", err)
			}
		}
	}

	// Populate legacy fields for backward compatibility
	user.Login = githubUser.Login
	user.GitHubToken = token

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &models.AuthResponse{
		Token: jwtToken,
		User:  *user,
	}, nil
}

// GitHubUserInfo represents GitHub user information
type GitHubUserInfo struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// getGitHubUserInfo gets user information from GitHub API
func (a *AuthServiceImpl) getGitHubUserInfo(ctx context.Context, token string) (*GitHubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API request failed with status: %d", resp.StatusCode)
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &GitHubUserInfo{
		ID:        fmt.Sprintf("%d", githubUser.ID),
		Login:     githubUser.Login,
		Email:     githubUser.Email,
		AvatarURL: githubUser.AvatarURL,
	}, nil
}

// Helper functions (moved to avoid duplication)
func authStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func authTimePtr(t time.Time) *time.Time {
	return &t
}

// exchangeCodeForToken exchanges authorization code for access token
func (a *AuthServiceImpl) exchangeCodeForToken(ctx context.Context, code string) (string, error) {
	// Prepare request data
	data := url.Values{}
	data.Set("client_id", a.config.GitHubOAuth.ClientID)
	data.Set("client_secret", a.config.GitHubOAuth.ClientSecret)
	data.Set("code", code)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub OAuth token exchange failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token received")
	}

	// Verify scopes (Requirements 1.1)
	requiredScopes := []string{"public_repo", "read:user", "user:email"}
	if !a.verifyScopesPresent(tokenResp.Scope, requiredScopes) {
		return "", fmt.Errorf("insufficient scopes granted: got %s, need %v", tokenResp.Scope, requiredScopes)
	}

	return tokenResp.AccessToken, nil
}

// verifyScopesPresent checks if all required scopes are present
func (a *AuthServiceImpl) verifyScopesPresent(grantedScopes string, requiredScopes []string) bool {
	scopes := strings.Split(grantedScopes, ",")
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[strings.TrimSpace(scope)] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}
	return true
}

// getUserInfo gets user information from GitHub API (legacy method)
func (a *AuthServiceImpl) getUserInfo(ctx context.Context, token string) (*models.User, error) {
	githubUser, err := a.getGitHubUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}

	// Get user email if not in profile (private email)
	email := githubUser.Email
	if email == "" {
		email, _ = a.getUserEmail(ctx, token) // Best effort, don't fail if email is private
	}

	return &models.User{
		ID:          githubUser.ID,
		Login:       githubUser.Login,
		Email:       email,
		GitHubToken: token,
	}, nil
}

// getUserEmail gets user email from GitHub API
func (a *AuthServiceImpl) getUserEmail(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API email request failed with status: %d", resp.StatusCode)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary email
	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	// Fallback to first email
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}
