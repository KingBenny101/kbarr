package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
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

// EnvAuth returns the credentials configured via environment and whether the
// environment override is active. When KBARR_AUTH_PASSWORD is set, kbarr
// authenticates solely against these values and the stored credentials are
// ignored — this is the declarative-config / lockout-recovery path: a forgotten
// password is fixed by editing the environment and restarting, never an
// unrecoverable state. The username defaults to "admin" when only the password
// is supplied, so a half-configured environment can't lock the user out.
func EnvAuth() (username, password string, active bool) {
	password = os.Getenv("KBARR_AUTH_PASSWORD")
	if password == "" {
		return "", "", false
	}
	username = strings.TrimSpace(os.Getenv("KBARR_AUTH_USERNAME"))
	if username == "" {
		username = "admin"
	}
	return username, password, true
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
	// Environment override wins outright: the stored hash is never consulted while
	// it is active. A plaintext compare is fine (and constant-time here) because
	// the env value is itself the secret — there is nothing to slow-hash.
	if envUser, envPass, active := EnvAuth(); active {
		return strings.EqualFold(username, envUser) &&
			subtle.ConstantTimeCompare([]byte(password), []byte(envPass)) == 1
	}
	storedUsername := config.Get(db, "authUsername", "Admin")
	storedHash := config.Get(db, "authPasswordHash", "")
	if !strings.EqualFold(username, storedUsername) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}

func UpdateCredentials(db *bun.DB, currentPassword, newUsername, newPassword string) error {
	// While the environment override is active the stored credentials are inert,
	// so changing them from the UI would be misleading — reject it outright.
	if _, _, active := EnvAuth(); active {
		return &ErrEnvManaged{}
	}
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

// ErrEnvManaged signals that credentials are pinned by the environment and
// cannot be changed through the API.
type ErrEnvManaged struct{}

func (e *ErrEnvManaged) Error() string { return "credentials are managed via environment variables" }
