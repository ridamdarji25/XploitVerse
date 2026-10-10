package pgapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xploitverse/backend/internal/config"
	"github.com/xploitverse/backend/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// Force imports to be recognized as used
var (
	_ *pgxpool.Pool
	_ *config.Config
)

func (a *API) Register(c *gin.Context) {
	var body struct {
		Username  string `json:"username" binding:"required,min=3,max=30"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	var existingCount int
	_ = a.DB.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) OR LOWER(username) = LOWER($2)
	`, strings.TrimSpace(body.Email), strings.TrimSpace(body.Username)).Scan(&existingCount)
	if existingCount > 0 {
		writeErr(c, http.StatusBadRequest, "User with this email or username already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	var id int64
	err = a.DB.QueryRow(c.Request.Context(), `
		INSERT INTO users (username, email, password_hash, role, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, strings.TrimSpace(body.Username), strings.ToLower(strings.TrimSpace(body.Email)), string(hash), roleStudent, strings.TrimSpace(body.FirstName), strings.TrimSpace(body.LastName)).Scan(&id)
	if err != nil {
		if pgUniqueViolation(err) {
			writeErr(c, http.StatusBadRequest, "User with this email or username already exists")
			return
		}
		writeErr(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	token, cookie, err := utils.CreateTokenResponse(strconv.FormatInt(id, 10), a.Cfg)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Registration successful",
		"data": gin.H{
			"user": gin.H{
				"id":        id,
				"username":  strings.TrimSpace(body.Username),
				"email":     strings.ToLower(strings.TrimSpace(body.Email)),
				"role":      roleStudent,
				"firstName": strings.TrimSpace(body.FirstName),
				"lastName":  strings.TrimSpace(body.LastName),
			},
			"token": token,
		},
	})
}

func (a *API) Login(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	var (
		id        int64
		username  string
		email     string
		passHash  string
		role      string
		firstName string
		lastName  string
		active    bool
	)
	err := a.DB.QueryRow(c.Request.Context(), `
		SELECT id, username, email, password_hash, role, first_name, last_name, is_active
		FROM users WHERE email = $1
	`, strings.ToLower(strings.TrimSpace(body.Email))).Scan(
		&id, &username, &email, &passHash, &role, &firstName, &lastName, &active,
	)
	if err != nil {
		writeErr(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if !active {
		writeErr(c, http.StatusUnauthorized, "User account has been deactivated")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(body.Password)); err != nil {
		writeErr(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	now := time.Now()
	_, _ = a.DB.Exec(c.Request.Context(), `UPDATE users SET last_login=$1, updated_at=$1 WHERE id=$2`, now, id)

	token, cookie, err := utils.CreateTokenResponse(strconv.FormatInt(id, 10), a.Cfg)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"data": gin.H{
			"user": gin.H{
				"id":        id,
				"username":  username,
				"email":     email,
				"role":      role,
				"firstName": firstName,
				"lastName":  lastName,
				"lastLogin": now,
			},
			"token": token,
		},
	})
}

func (a *API) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "jwt",
		Value:    "loggedout",
		Path:     "/",
		MaxAge:   10,
		HttpOnly: true,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logout successful"})
}

func (a *API) GetMe(c *gin.Context) {
	u := getAuthUser(c)
	if u == nil {
		writeErr(c, http.StatusUnauthorized, "Not authenticated")
		return
	}
	fullName := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if fullName == "" {
		fullName = u.Username
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": gin.H{
				"id":           u.ID,
				"username":     u.Username,
				"email":        u.Email,
				"role":         u.Role,
				"firstName":    u.FirstName,
				"lastName":     u.LastName,
				"fullName":     fullName,
				"totalLabTime": u.TotalLabTime,
				"totalSpent":   u.TotalSpent,
			},
		},
	})
}

func (a *API) UpdatePassword(c *gin.Context) {
	u := getAuthUser(c)
	if u == nil {
		writeErr(c, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var body struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	var currentHash string
	if err := a.DB.QueryRow(c.Request.Context(), `SELECT password_hash FROM users WHERE id=$1`, u.ID).Scan(&currentHash); err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(body.CurrentPassword)) != nil {
		writeErr(c, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 12)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	now := time.Now()
	_, err = a.DB.Exec(c.Request.Context(), `
		UPDATE users SET password_hash=$1, password_changed_at=$2, updated_at=$2 WHERE id=$3
	`, string(hash), now, u.ID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to update password")
		return
	}

	token, cookie, err := utils.CreateTokenResponse(strconv.FormatInt(u.ID, 10), a.Cfg)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password updated successfully", "data": gin.H{"token": token}})
}

func (a *API) RefreshToken(c *gin.Context) {
	u := getAuthUser(c)
	if u == nil {
		writeErr(c, http.StatusUnauthorized, "Not authenticated")
		return
	}
	token, cookie, err := utils.CreateTokenResponse(strconv.FormatInt(u.ID, 10), a.Cfg)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	http.SetCookie(c.Writer, cookie)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Token refreshed successfully", "data": gin.H{"token": token}})
}

func (a *API) ForgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, "Please provide a valid email address")
		return
	}

	var (
		userID   int64
		userName string
	)
	err := a.DB.QueryRow(c.Request.Context(), `
		SELECT id, username FROM users WHERE email=$1
	`, strings.ToLower(strings.TrimSpace(body.Email))).Scan(&userID, &userName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "If an account with that email exists, a password reset link has been sent.",
		})
		return
	}

	rawTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rawTokenBytes); err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate reset token")
		return
	}
	plainToken := hex.EncodeToString(rawTokenBytes)
	hash := sha256.Sum256([]byte(plainToken))
	hashedToken := hex.EncodeToString(hash[:])
	expiresAt := time.Now().Add(10 * time.Minute)

	if _, err := a.DB.Exec(c.Request.Context(), `
		UPDATE users SET password_reset_token=$1, password_reset_expires=$2, updated_at=now()
		WHERE id=$3
	`, hashedToken, expiresAt, userID); err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to persist reset token")
		return
	}

	clientURL := strings.TrimRight(a.Cfg.ClientURL, "/")
	resetURL := fmt.Sprintf("%s/reset-password/%s", clientURL, plainToken)

	if a.EmailSvc != nil && a.EmailSvc.IsConfigured() {
		go func(to, url, name string) {
			if err := a.EmailSvc.SendPasswordReset(to, url, name); err != nil {
				log.Printf("❌ Failed to send password reset email to %s: %v", to, err)
			}
		}(strings.TrimSpace(body.Email), resetURL, userName)
	} else {
		log.Printf("⚠️ SMTP not configured. Password reset link for %s: %s", body.Email, resetURL)
	}

	response := gin.H{
		"success": true,
		"message": "If an account with that email exists, a password reset link has been sent.",
	}

	c.JSON(http.StatusOK, response)
}

func (a *API) ResetPassword(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		writeErr(c, http.StatusBadRequest, "Reset token is required")
		return
	}

	var body struct {
		Password        string `json:"password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}
	if body.Password != body.ConfirmPassword {
		writeErr(c, http.StatusBadRequest, "Passwords do not match")
		return
	}

	hash := sha256.Sum256([]byte(token))
	hashedToken := hex.EncodeToString(hash[:])

	var userID int64
	err := a.DB.QueryRow(c.Request.Context(), `
		SELECT id FROM users
		WHERE password_reset_token=$1
			AND password_reset_expires IS NOT NULL
			AND password_reset_expires > now()
	`, hashedToken).Scan(&userID)
	if err != nil {
		writeErr(c, http.StatusBadRequest, "Token is invalid or has expired")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	now := time.Now()
	if _, err := a.DB.Exec(c.Request.Context(), `
		UPDATE users
		SET password_hash=$1,
			password_changed_at=$2,
			password_reset_token=NULL,
			password_reset_expires=NULL,
			updated_at=$2
		WHERE id=$3
	`, string(passwordHash), now, userID); err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to update password")
		return
	}

	jwtToken, cookie, err := utils.CreateTokenResponse(strconv.FormatInt(userID, 10), a.Cfg)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successful",
		"data":    gin.H{"token": jwtToken},
	})
}