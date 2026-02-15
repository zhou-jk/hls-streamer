package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Zhou-JK/hls-streamer/internal/config"
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepo
	cfg      config.JWTConfig
}

func NewAuthService(userRepo *repository.UserRepo, cfg config.JWTConfig) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *AuthService) Register(username, email, password string, roleID uint) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       roleID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return s.userRepo.FindByID(user.ID)
}

func (s *AuthService) Login(username, password string) (*TokenPair, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.Update(user)

	return s.generateTokenPair(user)
}

func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	stored, err := s.userRepo.FindRefreshToken(tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.userRepo.DeleteRefreshToken(tokenHash)
		return nil, errors.New("refresh token expired")
	}

	// Rotate: delete old, issue new
	_ = s.userRepo.DeleteRefreshToken(tokenHash)

	user, err := s.userRepo.FindByID(stored.UserID)
	if err != nil {
		return nil, err
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) Logout(refreshToken string) error {
	return s.userRepo.DeleteRefreshToken(hashToken(refreshToken))
}

func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("incorrect current password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	// Invalidate all refresh tokens
	return s.userRepo.DeleteUserRefreshTokens(userID)
}

func (s *AuthService) GetUser(id uint) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *AuthService) ListUsers(page, perPage int) ([]model.User, int64, error) {
	return s.userRepo.List(page, perPage)
}

func (s *AuthService) ListRoles() ([]model.Role, error) {
	return s.userRepo.ListRoles()
}

func (s *AuthService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

func (s *AuthService) UpdateUser(id uint, username, email *string, roleID *uint, isActive *bool) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if username != nil {
		user.Username = *username
	}
	if email != nil {
		user.Email = *email
	}
	if roleID != nil {
		user.RoleID = *roleID
	}
	if isActive != nil {
		user.IsActive = *isActive
	}
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(id)
}

func (s *AuthService) generateTokenPair(user *model.User) (*TokenPair, error) {
	roleName := ""
	if user.Role != nil {
		roleName = user.Role.Name
	}

	// Access token
	accessClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": roleName,
		"exp":  time.Now().Add(s.cfg.AccessTTL).Unix(),
		"iat":  time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token (opaque random string)
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, err
	}
	refreshStr := hex.EncodeToString(refreshBytes)

	// Store refresh token hash
	rt := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshStr),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTTL),
	}
	if err := s.userRepo.CreateRefreshToken(rt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int64(s.cfg.AccessTTL.Seconds()),
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
