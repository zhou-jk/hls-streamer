package repository

import (
	"github.com/Zhou-JK/hls-streamer/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Role").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Role").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Role").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) List(page, perPage int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	r.db.Model(&model.User{}).Count(&total)

	err := r.db.Preload("Role").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Order("id DESC").
		Find(&users).Error

	return users, total, err
}

func (r *UserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("is_active", false).Error
}

// Role operations

func (r *UserRepo) CreateRole(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *UserRepo) FindRoleByID(id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *UserRepo) FindRoleByName(name string) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *UserRepo) ListRoles() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Find(&roles).Error
	return roles, err
}

// Refresh token operations

func (r *UserRepo) CreateRefreshToken(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *UserRepo) FindRefreshToken(tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *UserRepo) DeleteRefreshToken(tokenHash string) error {
	return r.db.Where("token_hash = ?", tokenHash).Delete(&model.RefreshToken{}).Error
}

func (r *UserRepo) DeleteUserRefreshTokens(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.RefreshToken{}).Error
}
