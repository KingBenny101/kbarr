package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 30 * 24 * time.Hour

type session struct {
	username string
	expiry   time.Time
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]session
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]session)}
}

func (s *Store) Issue(username string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[token] = session{username: username, expiry: time.Now().Add(tokenTTL)}
	s.mu.Unlock()
	return token, nil
}

func (s *Store) Validate(token string) (username string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.expiry) {
		delete(s.sessions, token)
		return "", false
	}
	s.sessions[token] = session{username: sess.username, expiry: time.Now().Add(tokenTTL)}
	return sess.username, true
}

func (s *Store) Revoke(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		if _, ok := s.Validate(token); !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Store) UsernameFromRequest(r *http.Request) string {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	username, _ := s.Validate(token)
	return username
}

func EnsureDefaults(db *bun.DB) error {
	if config.Get(db, "authPasswordHash", "") != "" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := config.SetSetting(db, "authUsername", "Admin"); err != nil {
		return err
	}
	return config.SetSetting(db, "authPasswordHash", string(hash))
}

func ValidateCredentials(db *bun.DB, username, password string) bool {
	storedUsername := config.Get(db, "authUsername", "Admin")
	storedHash := config.Get(db, "authPasswordHash", "")
	if !strings.EqualFold(username, storedUsername) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}

func UpdateCredentials(db *bun.DB, currentPassword, newUsername, newPassword string) error {
	storedHash := config.Get(db, "authPasswordHash", "")
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(currentPassword)); err != nil {
		return &ErrUnauthorized{}
	}
	if newUsername != "" {
		if err := config.SetSetting(db, "authUsername", newUsername); err != nil {
			return err
		}
	}
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := config.SetSetting(db, "authPasswordHash", string(hash)); err != nil {
			return err
		}
	}
	return nil
}

type ErrUnauthorized struct{}

func (e *ErrUnauthorized) Error() string { return "invalid current password" }
