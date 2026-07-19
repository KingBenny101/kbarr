package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenTTL       = 30 * 24 * time.Hour
	absMaxTTL      = 90 * 24 * time.Hour
	cleanupInterval = 1 * time.Hour
)

type Store struct {
	db *bun.DB
}

func NewStore(db *bun.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Issue(username string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	hash := hashToken(token)
	now := time.Now()
	if _, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO sessions (token_hash, username, created_at, last_seen_at) VALUES (?, ?, ?, ?)`,
		hash, username, now, now,
	); err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return token, nil
}

func (s *Store) Validate(token string) (username string, ok bool) {
	hash := hashToken(token)
	var sess struct {
		Username   string    `bun:"username"`
		CreatedAt  time.Time `bun:"created_at"`
		LastSeenAt time.Time `bun:"last_seen_at"`
	}
	err := s.db.NewSelect().TableExpr("sessions").Column("username", "created_at", "last_seen_at").Where("token_hash = ?", hash).Scan(context.Background(), &sess)
	if err != nil {
		return "", false
	}

	now := time.Now()
	if now.After(sess.CreatedAt.Add(absMaxTTL)) || now.After(sess.LastSeenAt.Add(tokenTTL)) {
		s.Revoke(token)
		return "", false
	}

	if _, err := s.db.ExecContext(context.Background(), `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, now, hash); err != nil {
		return "", false
	}
	return sess.Username, true
}

func (s *Store) Revoke(token string) {
	hash := hashToken(token)
	s.db.ExecContext(context.Background(), `DELETE FROM sessions WHERE token_hash = ?`, hash)
}

func (s *Store) CleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.db.ExecContext(ctx,
				`DELETE FROM sessions WHERE last_seen_at < ? OR created_at < ?`,
				now.Add(-tokenTTL), now.Add(-absMaxTTL),
			)
		case <-ctx.Done():
			return
		}
	}
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

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

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

type ErrEnvManaged struct{}

func (e *ErrEnvManaged) Error() string { return "credentials are managed via environment variables" }
