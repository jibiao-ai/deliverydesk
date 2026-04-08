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
	if user.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
		if err != nil {
			return err
		}
		user.Password = string(hash)
	} else {
		user.Password = existing.Password
	}
	return repository.DB.Save(user).Error
}

func DeleteUser(id uint) error {
	return repository.DB.Delete(&model.User{}, id).Error
}
