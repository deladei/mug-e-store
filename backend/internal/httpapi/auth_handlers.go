package httpapi

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"coffeemug/backend/internal/auth"
	"coffeemug/backend/internal/store"
)

const (
	refreshCookieName = "refresh_token"
	refreshTTL        = 7 * 24 * time.Hour // 7-day rotating refresh token
)

// publicUser is the user shape returned to clients — never the password hash.
type publicUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func toPublicUser(u *store.User) publicUser {
	return publicUser{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone, Role: u.Role, CreatedAt: u.CreatedAt}
}

type authResponse struct {
	AccessToken string     `json:"access_token"`
	User        publicUser `json:"user"`
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, codeValidation, "name is required")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, codeValidation, "a valid email is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, codeValidation, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.serverError(w, "hashing password", err)
		return
	}
	u := &store.User{Name: req.Name, Email: req.Email, Phone: req.Phone, PasswordHash: hash}
	switch err := s.store.CreateUser(r.Context(), u); {
	case errors.Is(err, store.ErrEmailTaken):
		writeError(w, http.StatusConflict, codeEmailTaken, "that email is already registered")
		return
	case err != nil:
		s.serverError(w, "creating user", err)
		return
	}

	access, err := s.issueSession(w, r, u.ID)
	if err != nil {
		s.serverError(w, "issuing session", err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{AccessToken: access, User: toPublicUser(u)})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Identical response for unknown-email and wrong-password: no enumeration.
	u, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || auth.CheckPassword(u.PasswordHash, req.Password) != nil {
		writeError(w, http.StatusUnauthorized, codeInvalidCredentials, "invalid email or password")
		return
	}

	access, err := s.issueSession(w, r, u.ID)
	if err != nil {
		s.serverError(w, "issuing session", err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{AccessToken: access, User: toPublicUser(u)})
}

// handleRefresh rotates the refresh token (single-use) and returns a fresh
// access token. The cookie travels automatically.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "no session")
		return
	}
	hash := auth.HashRefreshToken(cookie.Value)
	rt, err := s.store.GetRefreshToken(r.Context(), hash)
	if err != nil {
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "session expired")
		return
	}

	// A used or expired token is destroyed; either way this token is now dead.
	_ = s.store.DeleteRefreshToken(r.Context(), hash)
	if time.Now().After(rt.ExpiresAt) {
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "session expired")
		return
	}

	u, err := s.store.GetUserByID(r.Context(), rt.UserID)
	if err != nil {
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "session expired")
		return
	}
	access, err := s.issueSession(w, r, u.ID)
	if err != nil {
		s.serverError(w, "issuing session", err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{AccessToken: access, User: toPublicUser(u)})
}

// handleLogout destroys the current refresh token and clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		_ = s.store.DeleteRefreshToken(r.Context(), auth.HashRefreshToken(cookie.Value))
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// issueSession creates a new rotating refresh token (stored hashed), sets it as
// an httpOnly cookie, and returns a fresh access token.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, userID int64) (string, error) {
	u, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		return "", err
	}
	raw, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(refreshTTL)
	if err := s.store.CreateRefreshToken(r.Context(), userID, auth.HashRefreshToken(raw), expiresAt); err != nil {
		return "", err
	}
	s.setRefreshCookie(w, raw, expiresAt)
	return s.tokens.GenerateAccessToken(userID, u.Role)
}

func (s *Server) setRefreshCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     "/api/v1/auth",
		Expires:  expires,
		HttpOnly: true,
		// Lax/!Secure suits same-site local dev; prod cross-origin switches to
		// SameSite=None; Secure (Session 5 / deploy).
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// serverError logs an unexpected error and returns a generic 500 (no internals
// leak to the client).
func (s *Server) serverError(w http.ResponseWriter, msg string, err error) {
	s.logger.Error(msg, "error", err)
	writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
}
