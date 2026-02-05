package model

import (
	"errors"
	"time"
)

// UserOAuthBinding stores the binding relationship between users and custom OAuth providers
type UserOAuthBinding struct {
	Id             int       `json:"id" gorm:"primaryKey"`
	UserId         int       `json:"user_id" gorm:"index;not null"`                                               // User ID
	ProviderId     int       `json:"provider_id" gorm:"index;not null"`                                           // Custom OAuth provider ID
	ProviderUserId string    `json:"provider_user_id" gorm:"type:varchar(256);not null"`                          // User ID from OAuth provider
	CreatedAt      time.Time `json:"created_at"`

	// Composite unique index to prevent duplicate bindings
	// One OAuth account can only be bound to one user
}

func (UserOAuthBinding) TableName() string {
	return "user_oauth_bindings"
}

// GetUserOAuthBindingsByUserId returns all OAuth bindings for a user
func GetUserOAuthBindingsByUserId(userId int) ([]*UserOAuthBinding, error) {
	var bindings []*UserOAuthBinding
	err := DB.Where("user_id = ?", userId).Find(&bindings).Error
	return bindings, err
}

// GetUserOAuthBinding returns a specific binding for a user and provider
func GetUserOAuthBinding(userId, providerId int) (*UserOAuthBinding, error) {
	var binding UserOAuthBinding
	err := DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// GetUserByOAuthBinding finds a user by provider ID and provider user ID
func GetUserByOAuthBinding(providerId int, providerUserId string) (*User, error) {
	var binding UserOAuthBinding
	err := DB.Where("provider_id = ? AND provider_user_id = ?", providerId, providerUserId).First(&binding).Error
	if err != nil {
		return nil, err
	}

	var user User
	err = DB.First(&user, binding.UserId).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// IsProviderUserIdTaken checks if a provider user ID is already bound to any user
func IsProviderUserIdTaken(providerId int, providerUserId string) bool {
	var count int64
	DB.Model(&UserOAuthBinding{}).Where("provider_id = ? AND provider_user_id = ?", providerId, providerUserId).Count(&count)
	return count > 0
}

// CreateUserOAuthBinding creates a new OAuth binding
func CreateUserOAuthBinding(binding *UserOAuthBinding) error {
	if binding.UserId == 0 {
		return errors.New("user ID is required")
	}
	if binding.ProviderId == 0 {
		return errors.New("provider ID is required")
	}
	if binding.ProviderUserId == "" {
		return errors.New("provider user ID is required")
	}

	// Check if this provider user ID is already taken
	if IsProviderUserIdTaken(binding.ProviderId, binding.ProviderUserId) {
		return errors.New("this OAuth account is already bound to another user")
	}

	binding.CreatedAt = time.Now()
	return DB.Create(binding).Error
}

// UpdateUserOAuthBinding updates an existing OAuth binding (e.g., rebind to different OAuth account)
func UpdateUserOAuthBinding(userId, providerId int, newProviderUserId string) error {
	// Check if the new provider user ID is already taken by another user
	var existingBinding UserOAuthBinding
	err := DB.Where("provider_id = ? AND provider_user_id = ?", providerId, newProviderUserId).First(&existingBinding).Error
	if err == nil && existingBinding.UserId != userId {
		return errors.New("this OAuth account is already bound to another user")
	}

	// Check if user already has a binding for this provider
	var binding UserOAuthBinding
	err = DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&binding).Error
	if err != nil {
		// No existing binding, create new one
		return CreateUserOAuthBinding(&UserOAuthBinding{
			UserId:         userId,
			ProviderId:     providerId,
			ProviderUserId: newProviderUserId,
		})
	}

	// Update existing binding
	return DB.Model(&binding).Update("provider_user_id", newProviderUserId).Error
}

// DeleteUserOAuthBinding deletes an OAuth binding
func DeleteUserOAuthBinding(userId, providerId int) error {
	return DB.Where("user_id = ? AND provider_id = ?", userId, providerId).Delete(&UserOAuthBinding{}).Error
}

// DeleteUserOAuthBindingsByUserId deletes all OAuth bindings for a user
func DeleteUserOAuthBindingsByUserId(userId int) error {
	return DB.Where("user_id = ?", userId).Delete(&UserOAuthBinding{}).Error
}

// GetBindingCountByProviderId returns the number of bindings for a provider
func GetBindingCountByProviderId(providerId int) (int64, error) {
	var count int64
	err := DB.Model(&UserOAuthBinding{}).Where("provider_id = ?", providerId).Count(&count).Error
	return count, err
}
