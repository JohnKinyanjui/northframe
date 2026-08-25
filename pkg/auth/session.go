// Package auth provides optional server-side sessions and permission guards
// for Northframe applications without prescribing a user model or database.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

// Session contains authentication state stored on the server. The browser
// only receives a random opaque token; raw tokens are never persisted.
type Session struct {
	ID          string
	Subject     string
	Values      map[string]string
	Permissions []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type SessionInput struct {
	Subject     string
	Values      map[string]string
	Permissions []string
}

// Store persists sessions by their SHA-256 token digest.
type Store interface {
	Load(context.Context, string) (Session, error)
	Save(context.Context, Session) error
	Delete(context.Context, string) error
}

type Config struct {
	Store      Store
	CookieName string
	CookiePath string
	Domain     string
	Lifetime   time.Duration
	Secure     bool
	SameSite   http.SameSite
	Clock      func() time.Time
	Random     io.Reader
}

// Manager creates and resolves opaque server-side sessions.
type Manager struct {
	store      Store
	cookieName string
	cookiePath string
	domain     string
	lifetime   time.Duration
	secure     bool
	sameSite   http.SameSite
	clock      func() time.Time
	random     io.Reader
}

func New(config Config) (*Manager, error) {
	if config.Store == nil {
		return nil, errors.New("auth session store is required")
	}
	if strings.TrimSpace(config.CookieName) == "" {
		config.CookieName = "northframe_session"
	}
	if strings.TrimSpace(config.CookiePath) == "" {
		config.CookiePath = "/"
	}
	if config.Lifetime <= 0 {
		config.Lifetime = 24 * time.Hour
	}
	if config.SameSite == 0 {
		config.SameSite = http.SameSiteLaxMode
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	return &Manager{
		store:      config.Store,
		cookieName: config.CookieName,
		cookiePath: config.CookiePath,
		domain:     config.Domain,
		lifetime:   config.Lifetime,
		secure:     config.Secure,
		sameSite:   config.SameSite,
		clock:      config.Clock,
		random:     config.Random,
	}, nil
}

// Start creates a new session and writes its opaque token cookie.
func (manager *Manager) Start(ctx context.Context, writer http.ResponseWriter, input SessionInput) (Session, error) {
	if strings.TrimSpace(input.Subject) == "" {
		return Session{}, errors.New("session subject is required")
	}
	rawToken := make([]byte, 32)
	if _, err := io.ReadFull(manager.random, rawToken); err != nil {
		return Session{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	now := manager.clock().UTC()
	session := Session{
		ID:          tokenDigest(token),
		Subject:     input.Subject,
		Values:      cloneValues(input.Values),
		Permissions: normalizePermissions(input.Permissions),
		CreatedAt:   now,
		ExpiresAt:   now.Add(manager.lifetime),
	}
	if err := manager.store.Save(ctx, session); err != nil {
		return Session{}, fmt.Errorf("save session: %w", err)
	}
	http.SetCookie(writer, manager.cookie(token, session.ExpiresAt, int(manager.lifetime.Seconds())))
	return cloneSession(session), nil
}

// Get resolves the current request's session.
func (manager *Manager) Get(ctx context.Context, request *http.Request) (Session, error) {
	cookie, err := request.Cookie(manager.cookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return Session{}, ErrSessionNotFound
	}
	session, err := manager.store.Load(ctx, tokenDigest(cookie.Value))
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("load session: %w", err)
	}
	if !session.ExpiresAt.After(manager.clock()) {
		_ = manager.store.Delete(ctx, session.ID)
		return Session{}, ErrSessionNotFound
	}
	return cloneSession(session), nil
}

// End revokes the current session and expires its browser cookie.
func (manager *Manager) End(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	cookie, err := request.Cookie(manager.cookieName)
	if err == nil && cookie.Value != "" {
		if err := manager.store.Delete(ctx, tokenDigest(cookie.Value)); err != nil && !errors.Is(err, ErrSessionNotFound) {
			return fmt.Errorf("delete session: %w", err)
		}
	}
	http.SetCookie(writer, manager.cookie("", time.Unix(1, 0), -1))
	return nil
}

// Save updates application values and permissions without exposing the token.
func (manager *Manager) Save(ctx context.Context, session Session) error {
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.Subject) == "" {
		return errors.New("session ID and subject are required")
	}
	if !session.ExpiresAt.After(manager.clock()) {
		return errors.New("cannot save an expired session")
	}
	session.Values = cloneValues(session.Values)
	session.Permissions = normalizePermissions(session.Permissions)
	if err := manager.store.Save(ctx, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (manager *Manager) cookie(value string, expires time.Time, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: manager.cookieName, Value: value, Path: manager.cookiePath,
		Domain: manager.domain, Expires: expires, MaxAge: maxAge,
		Secure: manager.secure, HttpOnly: true, SameSite: manager.sameSite,
	}
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func cloneValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneSession(session Session) Session {
	session.Values = cloneValues(session.Values)
	session.Permissions = append([]string(nil), session.Permissions...)
	return session
}
