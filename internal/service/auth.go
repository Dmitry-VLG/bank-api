package service

import (
	"context"
	"errors"
	"strings"

	"bank-api/internal/model"
	"bank-api/internal/repository"
	"bank-api/internal/security"
)

type AuthService struct {
	store  *repository.Store
	tokens security.TokenManager
}

type RegisterInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User  model.User `json:"user"`
	Token string     `json:"token"`
}

func NewAuthService(store *repository.Store, tokens security.TokenManager) *AuthService {
	return &AuthService{
		store:  store,
		tokens: tokens,
	}
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthResponse, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Username = strings.TrimSpace(in.Username)

	if !model.ValidateEmail(in.Email) || !model.ValidateUsername(in.Username) || !model.ValidatePassword(in.Password) {
		return AuthResponse{}, ErrInvalidInput
	}

	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return AuthResponse{}, err
	}

	u, err := s.store.CreateUser(ctx, in.Email, in.Username, hash)
	if errors.Is(err, repository.ErrConflict) {
		return AuthResponse{}, ErrConflict
	}
	if err != nil {
		return AuthResponse{}, err
	}

	token, err := s.tokens.Generate(u.ID)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:  u,
		Token: token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if !model.ValidateEmail(email) || in.Password == "" {
		return AuthResponse{}, ErrInvalidInput
	}

	u, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return AuthResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResponse{}, err
	}

	if err := security.VerifyPassword(u.PasswordHash, in.Password); err != nil {
		return AuthResponse{}, ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(u.ID)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:  u,
		Token: token,
	}, nil
}
