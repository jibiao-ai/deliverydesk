package service

import (
	"errors"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers() ([]model.User, error) {
	var users []model.User
	err := repository.DB.Find(&users).Error
	return users, err
}

func GetUserByID(id uint, user *model.User) error {
	return repository.DB.First(user, id).Error
}

func CreateUser(user *model.User) error {
	if user.Username == "" {
		return errors.New("username is required")
	}
	if user.Role == "" {
		user.Role = "user"
	}
	if user.AuthType == "" {
		user.AuthType = "local"
	}
	if user.AuthType == "local" && user.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
		if err != nil {
			return err
		}
		user.Password = string(hash)
	}
	return repository.DB.Create(user).Error
}

func UpdateUser(user *model.User) error {
	existing := model.User{}
	if err := repository.DB.First(&existing, user.ID).Error; err != nil {
		return errors.New("user not found")
	}
	// Preserve auth_type - it cannot be changed via update
	user.AuthType = existing.AuthType

	// Build update map — only update fields that are provided.
	// Using a map with db.Model().Updates() avoids overwriting created_at
	// with zero value (which causes MySQL Error 1292 on STRICT mode).
	updates := map[string]interface{}{
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"auth_type":    existing.AuthType,
	}
	if user.Username != "" {
		updates["username"] = user.Username
	}
	if user.Password != "" && existing.AuthType == "local" {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
		if err != nil {
			return err
		}
		updates["password"] = string(hash)
	}
	return repository.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func DeleteUser(id uint) error {
	return repository.DB.Delete(&model.User{}, id).Error
}
