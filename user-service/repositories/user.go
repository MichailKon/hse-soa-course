package repositories

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"user-service/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return r.db.Create(user).Error
}

func userFromDbResponse(fetchedUser *models.User, response *gorm.DB) (*models.User, error) {
	if response.Error != nil {
		if errors.Is(response.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, response.Error
	}
	return fetchedUser, nil
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	return userFromDbResponse(&user, r.db.Where(&models.User{Username: username}).First(&user))
}

func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	return userFromDbResponse(&user, r.db.First(&user, id))
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	return userFromDbResponse(&user, r.db.Where(&models.User{Email: email}).First(&user))
}

func (r *UserRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}
