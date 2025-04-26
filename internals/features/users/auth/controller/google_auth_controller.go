package controller

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"masjidku/internals/configs"
	"masjidku/internals/features/users/user/models"
)

// GoogleConfig holds the configuration for Google OAuth
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// GoogleUserInfo holds user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// TokenResponse holds the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// UserResponse is the clean data structure sent to the frontend
type UserResponse struct {
	ID           uuid.UUID `json:"id"` 
	UserName     string    `json:"user_name"`
	Email        string    `json:"email"`
	GoogleID     *string   `json:"google_id,omitempty"`
	Role         string    `json:"role"`
	DonationName *string   `json:"donation_name,omitempty"`
	OriginalName *string   `json:"original_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuthResponse is the response sent to the frontend after successful authentication
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// GoogleAuthController handles Google authentication
type GoogleAuthController struct {
	DB           *gorm.DB
	GoogleConfig GoogleConfig
}

// NewGoogleAuthController creates a new GoogleAuthController
func NewGoogleAuthController(db *gorm.DB) *GoogleAuthController {
	return &GoogleAuthController{
		DB: db,
		GoogleConfig: GoogleConfig{
			ClientID:     configs.GetEnv("GOOGLE_CLIENT_ID"),
			ClientSecret: configs.GetEnv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  configs.GetEnv("GOOGLE_REDIRECT_URL"),
			Scopes:       []string{"profile", "email"},
			AuthURL:      "https://accounts.google.com/o/oauth2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		},
	}
}

// GoogleLogin initiates the Google OAuth flow
func (gc *GoogleAuthController) GoogleLogin(c *fiber.Ctx) error {
	log.Println("[INFO] Starting Google Login process")

	// Generate cryptographically secure random state
	state, err := generateRandomString(32)
	if err != nil {
		log.Printf("[ERROR] Failed to generate random state: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication initialization failed",
		})
	}

	// Store state in session
	c.Cookie(&fiber.Cookie{
		Name:     "google_oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "lax",
		Path:     "/",
	})

	// Build the authorization URL
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		gc.GoogleConfig.AuthURL,
		gc.GoogleConfig.ClientID,
		url.QueryEscape(gc.GoogleConfig.RedirectURL),
		url.QueryEscape("email profile"),
		state,
	)

	log.Printf("[INFO] Redirecting to Google auth URL: %s", authURL)
	return c.Redirect(authURL)
}

// GoogleCallback handles the callback from Google
func (gc *GoogleAuthController) GoogleCallback(c *fiber.Ctx) error {
	log.Println("[INFO] Handling Google callback")

	// 1. Validate the incoming request
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		log.Println("[ERROR] No code received from Google!")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No code received from Google",
		})
	}

	// Verify state
	storedState := c.Cookies("google_oauth_state")
	if storedState == "" || storedState != state {
		log.Println("[ERROR] Invalid state parameter")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid state parameter",
		})
	}

	// Clear the state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "google_oauth_state",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "lax",
		Path:     "/",
	})

	// 2. Exchange code for token
	tokenResponse, err := gc.exchangeCodeForToken(code)
	if err != nil {
		log.Printf("[ERROR] Token exchange failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to authenticate with Google",
		})
	}

	// 3. Get user info
	userInfo, err := gc.getUserInfo(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("[ERROR] Failed to get user info: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	// 4. Process user data and create/update user in database
	user, err := gc.processUserData(userInfo)
	if err != nil {
		log.Printf("[ERROR] Failed to process user data: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process user account",
		})
	}

	// 5. Generate JWT token
	token, err := gc.generateJWTToken(user)
	if err != nil {
		log.Printf("[ERROR] Failed to generate token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication token",
		})
	}

	// 6. Prepare and return clean response
	response := AuthResponse{
		Token: token,
		User: UserResponse{
			ID:           user.ID,
			UserName:     user.UserName,
			Email:        user.Email,
			GoogleID:     user.GoogleID,
			Role:         user.Role,
			DonationName: user.DonationName,
			OriginalName: user.OriginalName,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
		},
	}

	log.Printf("[SUCCESS] Google login successful for user: ID=%d, Email=%s", user.ID, user.Email)
	return c.JSON(response)
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (gc *GoogleAuthController) exchangeCodeForToken(code string) (*TokenResponse, error) {
	tokenData := url.Values{}
	tokenData.Set("code", code)
	tokenData.Set("client_id", gc.GoogleConfig.ClientID)
	tokenData.Set("client_secret", gc.GoogleConfig.ClientSecret)
	tokenData.Set("redirect_uri", gc.GoogleConfig.RedirectURL)
	tokenData.Set("grant_type", "authorization_code")

	tokenReq, err := http.NewRequest("POST", gc.GoogleConfig.TokenURL,
		strings.NewReader(tokenData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	tokenBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(tokenBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, errors.New("no access token received from Google")
	}

	return &tokenResp, nil
}

// getUserInfo fetches user information using the access token
func (gc *GoogleAuthController) getUserInfo(accessToken string) (*GoogleUserInfo, error) {
	userReq, err := http.NewRequest("GET", gc.GoogleConfig.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	userReq.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute user info request: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", userResp.StatusCode, string(body))
	}

	userData, err := io.ReadAll(userResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(userData, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Validate essential user info
	if userInfo.ID == "" || userInfo.Email == "" {
		return nil, errors.New("invalid user info from Google")
	}

	return &userInfo, nil
}

// processUserData processes the user information and creates or updates the user in the database
func (gc *GoogleAuthController) processUserData(userInfo *GoogleUserInfo) (*models.UserModel, error) {
	var user models.UserModel
	tx := gc.DB.Begin()

	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Try to find user by Google ID
	result := tx.Where("google_id = ?", userInfo.ID).First(&user)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, fmt.Errorf("database error when searching by Google ID: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// User not found by Google ID, try to find by email
		result = tx.Where("email = ?", userInfo.Email).First(&user)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, fmt.Errorf("database error when searching by email: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			// User not found, create new user
			log.Printf("[INFO] Creating new user for Google ID: %s, Email: %s", userInfo.ID, userInfo.Email)

			// Generate random password
			randomPassword, err := generateRandomString(16)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to generate random password: %w", err)
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to hash password: %w", err)
			}

			// Setup user data
			googleID := userInfo.ID
			userName := userInfo.Name
			if userName == "" {
				userName = strings.Split(userInfo.Email, "@")[0]
			}

			newUser := models.UserModel{
				UserName:         userName,
				Email:            userInfo.Email,
				Password:         string(hashedPassword),
				GoogleID:         &googleID,
				Role:             "user",
				SecurityQuestion: "Account created with Google Auth",
				SecurityAnswer:   "google_auth_user",
				OriginalName:     &userName,
			}

			// Create new user
			if err := tx.Create(&newUser).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			// Commit the transaction
			if err := tx.Commit().Error; err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}

			// Retrieve the newly created user to ensure complete data
			if err := gc.DB.First(&user, newUser.ID).Error; err != nil {
				return nil, fmt.Errorf("failed to retrieve newly created user: %w", err)
			}

			log.Printf("[SUCCESS] New user created: ID=%d, Email=%s", user.ID, user.Email)
		} else {
			// User found by email, update Google ID
			log.Printf("[INFO] Updating existing user with Google ID: %s", userInfo.ID)
			googleID := userInfo.ID
			user.GoogleID = &googleID

			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to update user with Google ID: %w", err)
			}

			if err := tx.Commit().Error; err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}

			log.Printf("[SUCCESS] Updated user with Google ID: ID=%d, Email=%s", user.ID, user.Email)
		}
	} else {
		// User found by Google ID
		log.Printf("[INFO] User found by Google ID: %s", userInfo.ID)

		// Update user info if needed
		needsUpdate := false
		if user.Email != userInfo.Email {
			user.Email = userInfo.Email
			needsUpdate = true
		}

		// Update user name if it's empty
		if user.UserName == "" && userInfo.Name != "" {
			user.UserName = userInfo.Name
			needsUpdate = true
		}

		if needsUpdate {
			if err := tx.Save(&user).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to update user: %w", err)
			}

			if err := tx.Commit().Error; err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}

			log.Printf("[SUCCESS] Updated user: ID=%d, Email=%s", user.ID, user.Email)
		} else {
			// No changes, commit transaction
			if err := tx.Commit().Error; err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}

	return &user, nil
}

// generateJWTToken generates a JWT token for the user
func (gc *GoogleAuthController) generateJWTToken(user *models.UserModel) (string, error) {
	expirationTime := time.Now().Add(time.Hour * 96) // 4 days
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   expirationTime.Unix(),
		"iat":   time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(configs.GetEnv("JWT_SECRET")))
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}
