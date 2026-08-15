package model

import (
	"errors"

	"gorm.io/gorm"
)

// ErrSetupAlreadyInitialized 表示 setup 记录已被其他调用方（并发请求或多实例）
// 率先写入，当前调用应按"系统已初始化"处理。
var ErrSetupAlreadyInitialized = errors.New("setup already initialized")

// Setup 的 ID 固定为 setupRecordID：全库只允许一条初始化记录，
// 主键唯一约束（SQLite/MySQL/PostgreSQL 一致支持）作为跨实例初始化的原子占位。
const setupRecordID = 1

type Setup struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	Version       string `json:"version" gorm:"type:varchar(50);not null"`
	InitializedAt int64  `json:"initialized_at" gorm:"type:bigint;not null"`
}

func GetSetup() *Setup {
	var setup Setup
	err := DB.First(&setup).Error
	if err != nil {
		return nil
	}
	return &setup
}

// NewSetupRecord 构造带固定主键的初始化记录。占用固定主键使并发插入
// 在数据库层面串行化：只有一个调用方能成功，其余按已初始化处理。
func NewSetupRecord(version string, initializedAt int64) Setup {
	return Setup{
		ID:            setupRecordID,
		Version:       version,
		InitializedAt: initializedAt,
	}
}

// InitializeSetup 在单个数据库事务内写入 setup 记录并按需创建 root 用户。
//
// 多实例部署下，进程内锁无法阻止不同进程同时通过 constant.Setup /
// RootUserExists 检查后交错写入（重复 root 用户、重复 setup 记录）。
// 事务内先插入固定主键的 setup 记录占位：
//   - 并发实例的占位插入因主键冲突失败，事务回滚后按已初始化返回；
//   - root 用户创建失败时整个事务回滚，不会出现"已初始化但无 root"的中间态。
func InitializeSetup(rootUser *User, setup Setup) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&setup).Error; err != nil {
			return err
		}
		if rootUser != nil {
			if err := tx.Create(rootUser).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		return nil
	}
	// 区分"占位冲突（已被其他实例初始化）"与真实数据库故障：
	// 冲突意味着 setup 记录已可查到（或 root 已存在的历史状态），
	// 此时按已初始化返回；否则向上暴露原始错误。
	if GetSetup() != nil {
		return ErrSetupAlreadyInitialized
	}
	if rootUser != nil && RootUserExists() {
		return ErrSetupAlreadyInitialized
	}
	return err
}
