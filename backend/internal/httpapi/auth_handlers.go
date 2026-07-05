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
	passwordResetTTL  = time.Hour          // a reset link is valid for one hour
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

// guestEmailDomain namespaces the synthetic, non-routable emails minted for
// guest accounts so they can never collide with a real user's address.
const guestEmailDomain = "guest.coffeemug.local"

type guestRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// handleGuestSession mints a passwordless guest account and returns a normal
// session (PRD S4). The guest IS a customer row — so the cart, checkout,
// ownership, order tracking and history paths all work unchanged downstream —
// but it carries is_guest=true, a synthetic unique email, and an unusable random
// password hash, so no one can ever log in as it and it earns no loyalty points.
//
// name and phone are optional: the guest-checkout form collects them up front so
// staff have a name to call out and a number to reach, but an anonymous guest
// (no body) is allowed and shows as "Guest". The order's own address/phone still
// come from POST /checkout, so the shared checkout contract is untouched.
func (s *Server) handleGuestSession(w http.ResponseWriter, r *http.Request) {
	var req guestRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "Guest"
	}

	// A synthetic, unique, non-routable email keeps the UNIQUE/NOT NULL email
	// contract intact without touching the schema; the token makes a collision
	// statistically impossible.
	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		s.serverError(w, "generating guest id", err)
		return
	}
	email := "guest-" + token + "@" + guestEmailDomain

	// An unusable password: bcrypt of a fresh random secret no one holds. The
	// constant-time login check can therefore never succeed for a guest, even if
	// its synthetic email were somehow guessed.
	secret, err := auth.GenerateOpaqueToken()
	if err != nil {
		s.serverError(w, "generating guest secret", err)
		return
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		s.serverError(w, "hashing guest secret", err)
		return
	}

	u := &store.User{Name: req.Name, Email: email, Phone: strings.TrimSpace(req.Phone), PasswordHash: hash, IsGuest: true}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		s.serverError(w, "creating guest", err)
		return
	}

	access, err := s.issueSession(w, r, u.ID)
	if err != nil {
		s.serverError(w, "issuing guest session", err)
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

type passwordResetRequestRequest struct {
	Email string `json:"email"`
}

// handlePasswordResetRequest starts a reset. It ALWAYS responds 200 with the
// same generic body whether or not the email is registered — revealing "no such
// account" here would turn this into an account-enumeration oracle (CLAUDE.md
// §4). When the email does resolve to a user, an opaque token is generated, its
// hash stored with a one-hour expiry, and the raw token handed to delivery.
//
// Delivery is out of band: there is no email provider wired (no SMTP dep, per
// §5), so the reset link is logged server-side for the demo/E2E. Production
// swaps this single call for a real mailer — see DECISIONS.md (2026-06-14).
func (s *Server) handlePasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequestRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Generic success regardless of outcome. Built once and reused on every path.
	ok := func() {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "if that email is registered, a password reset link has been sent",
		})
	}

	// A malformed email can never match a user; respond generically, do no work.
	if _, err := mail.ParseAddress(req.Email); err != nil {
		ok()
		return
	}

	u, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) {
		ok()
		return
	}
	if err != nil {
		s.serverError(w, "password reset lookup", err)
		return
	}

	raw, err := auth.GenerateOpaqueToken()
	if err != nil {
		s.serverError(w, "generating reset token", err)
		return
	}
	if err := s.store.CreatePasswordResetToken(r.Context(), u.ID, auth.HashToken(raw), time.Now().Add(passwordResetTTL)); err != nil {
		s.serverError(w, "storing reset token", err)
		return
	}

	// TODO(deploy): replace this log with a real email send to u.Email.
	resetLink := strings.TrimRight(s.cfg.FrontendOrigin, "/") + "/reset-password?token=" + raw
	s.logger.Info("password reset requested", "user_id", u.ID, "reset_link", resetLink)

	ok()
}

type passwordResetConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// handlePasswordResetConfirm completes a reset: it validates the new password,
// then atomically redeems the token and sets the password (store.ConsumePasswordReset
// also clears the user's other reset tokens and revokes all their sessions). A
// token that is unknown, already used, or expired all yield the same 400
// invalid_token — the client cannot tell which, and should just request a new link.
func (s *Server) handlePasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, codeInvalidToken, "a reset token is required")
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

	switch err := s.store.ConsumePasswordReset(r.Context(), auth.HashToken(req.Token), hash); {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, store.ErrTokenExpired):
		writeError(w, http.StatusBadRequest, codeInvalidToken, "this reset link is invalid or has expired")
		return
	case err != nil:
		s.serverError(w, "consuming reset token", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "your password has been updated"})
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
		// Lax/!Secure suits same-site local dev; COOKIE_SECURE=true switches to
		// SameSite=None; Secure, without which the browser drops the cookie
		// when the frontend and the API live on different domains.
		SameSite: s.cookieSameSite(),
		Secure:   s.cfg.CookieSecure,
	})
}

func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: s.cookieSameSite(),
		Secure:   s.cfg.CookieSecure,
	})
}

// cookieSameSite pairs None with Secure: SameSite=None is only valid on a
// Secure cookie, so the two attributes must flip together.
func (s *Server) cookieSameSite() http.SameSite {
	if s.cfg.CookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

// serverError logs an unexpected error and returns a generic 500 (no internals
// leak to the client).
func (s *Server) serverError(w http.ResponseWriter, msg string, err error) {
	s.logger.Error(msg, "error", err)
	writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
}
