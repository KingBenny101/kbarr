package handlers

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/core/auth"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

func Login(store *auth.Store) func(context.Context, *LoginInput) (*TokenOutput, error) {
	return func(ctx context.Context, input *LoginInput) (*TokenOutput, error) {
		if !auth.ValidateCredentials(db.DB, input.Body.Username, input.Body.Password) {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}
		token, err := store.Issue(input.Body.Username)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to issue token", err)
		}
		return &TokenOutput{Body: TokenResponse{Token: token}}, nil
	}
}

func Logout(store *auth.Store) func(context.Context, *LogoutInput) (*struct{}, error) {
	return func(ctx context.Context, input *LogoutInput) (*struct{}, error) {
		token := strings.TrimPrefix(input.Authorization, "Bearer ")
		store.Revoke(token)
		return nil, nil
	}
}

func Me(store *auth.Store) func(context.Context, *MeInput) (*UsernameOutput, error) {
	return func(ctx context.Context, input *MeInput) (*UsernameOutput, error) {
		token := strings.TrimPrefix(input.Authorization, "Bearer ")
		username, _ := store.Validate(token)
		if username == "" {
			username = config.Get(db.DB, "authUsername", "admin")
		}
		return &UsernameOutput{Body: UsernameResponse{Username: username}}, nil
	}
}

func ChangeCredentials() func(context.Context, *ChangeCredentialsInput) (*struct{}, error) {
	return func(ctx context.Context, input *ChangeCredentialsInput) (*struct{}, error) {
		err := auth.UpdateCredentials(db.DB, input.Body.CurrentPassword, input.Body.NewUsername, input.Body.NewPassword)
		if err != nil {
			if _, ok := err.(*auth.ErrUnauthorized); ok {
				return nil, huma.Error401Unauthorized("invalid current password")
			}
			return nil, huma.Error500InternalServerError("failed to update credentials", err)
		}
		return nil, nil
	}
}
