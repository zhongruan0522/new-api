package userstore

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/infra/redis"
	"github.com/NookMux/NookMux/internal/infra/runtime"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
	"time"
)

// UserBase struct remains the same as it represents the cached data structure
type UserBase struct {
	Id       int    `json:"id"`
	Group    string `json:"group"`
	Email    string `json:"email"`
	Quota    int    `json:"quota"`
	Status   int    `json:"status"`
	Role     int    `json:"role"`
	Username string `json:"username"`
	Setting  string `json:"setting"`
}

func (user *UserBase) WriteContext(c *gin.Context) {
	httpapi.SetContextKey(c, common.ContextKeyUserGroup, user.Group)
	httpapi.SetContextKey(c, common.ContextKeyUserQuota, user.Quota)
	httpapi.SetContextKey(c, common.ContextKeyUserStatus, user.Status)
	httpapi.SetContextKey(c, common.ContextKeyUserEmail, user.Email)
	httpapi.SetContextKey(c, common.ContextKeyUserName, user.Username)
	httpapi.SetContextKey(c, common.ContextKeyUserSetting, user.GetSetting())
}

func (user *UserBase) GetSetting() shared.UserSetting {
	setting := shared.UserSetting{}
	if user.Setting != "" {
		err := jsonx.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

// getUserCacheKey returns the key for user cache
func getUserCacheKey(userId int) string {
	return fmt.Sprintf("user:%d", userId)
}

// invalidateUserCache clears user cache
func invalidateUserCache(userId int) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisDelKey(getUserCacheKey(userId))
}

// InvalidateUserCache clears the Redis user cache for security-sensitive updates.
func InvalidateUserCache(userId int) error {
	return invalidateUserCache(userId)
}

// updateUserCache updates all user cache fields using hash
func updateUserCache(user User) error {
	if !redis.RedisEnabled {
		return nil
	}

	return redis.RedisHSetObj(
		getUserCacheKey(user.Id),
		user.ToBaseUser(),
		time.Duration(redis.RedisKeyCacheSeconds())*time.Second,
	)
}

// GetUserCache gets complete user cache from hash
func GetUserCache(userId int) (userCache *UserBase, err error) {
	var user *User
	var fromDB bool
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if dbstore.ShouldUpdateRedis(fromDB, err) && user != nil {
			runtime.RelayGo(func() {
				if err := updateUserCache(*user); err != nil {
					common.SysLog("failed to update user status cache: " + err.Error())
				}
			})
		}
	}()

	// Try getting from Redis first
	userCache, err = cacheGetUserBase(userId)
	if err == nil {
		return userCache, nil
	}

	// If Redis fails, get from DB
	fromDB = true
	user, err = GetUserById(userId, false)
	if err != nil {
		return nil, err // Return nil and error if DB lookup fails
	}

	// Create cache object from user data
	userCache = &UserBase{
		Id:       user.Id,
		Group:    user.Group,
		Quota:    user.Quota,
		Status:   user.Status,
		Role:     user.Role,
		Username: user.Username,
		Setting:  user.Setting,
		Email:    user.Email,
	}

	return userCache, nil
}

func cacheGetUserBase(userId int) (*UserBase, error) {
	if !redis.RedisEnabled {
		return nil, fmt.Errorf("redis is not enabled")
	}
	var userCache UserBase
	// Try getting from Redis first
	err := redis.RedisHGetObj(getUserCacheKey(userId), &userCache)
	if err != nil {
		return nil, err
	}
	return &userCache, nil
}

// Add atomic quota operations using hash fields
func CacheIncrUserQuota(userId int, delta int64) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisHIncrBy(getUserCacheKey(userId), "Quota", delta)
}

func cacheDecrUserQuota(userId int, delta int64) error {
	return CacheIncrUserQuota(userId, -delta)
}

// Helper functions to get individual fields if needed
func getUserGroupCache(userId int) (string, error) {
	cache, err := GetUserCache(userId)
	if err != nil {
		return "", err
	}
	return cache.Group, nil
}

func getUserQuotaCache(userId int) (int, error) {
	cache, err := GetUserCache(userId)
	if err != nil {
		return 0, err
	}
	return cache.Quota, nil
}

func getUserNameCache(userId int) (string, error) {
	cache, err := GetUserCache(userId)
	if err != nil {
		return "", err
	}
	return cache.Username, nil
}

func getUserSettingCache(userId int) (shared.UserSetting, error) {
	cache, err := GetUserCache(userId)
	if err != nil {
		return shared.UserSetting{}, err
	}
	return cache.GetSetting(), nil
}

// New functions for individual field updates
func updateUserQuotaCache(userId int, quota int) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisHSetField(getUserCacheKey(userId), "Quota", fmt.Sprintf("%d", quota))
}

func updateUserGroupCache(userId int, group string) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisHSetField(getUserCacheKey(userId), "Group", group)
}

func UpdateUserGroupCache(userId int, group string) error {
	return updateUserGroupCache(userId, group)
}

func updateUserNameCache(userId int, username string) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisHSetField(getUserCacheKey(userId), "Username", username)
}

func updateUserSettingCache(userId int, setting string) error {
	if !redis.RedisEnabled {
		return nil
	}
	return redis.RedisHSetField(getUserCacheKey(userId), "Setting", setting)
}
