package model

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupSetupTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Setup{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
	})
}

func newSetupRootUser(name string) *User {
	return &User{
		Username:    name,
		Password:    "hashed-password",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Root User",
		AccessToken: nil,
		Quota:       100000000,
		AffCode:     "aff-" + name,
	}
}

// 多实例并发初始化：两个进程同时通过 constant.Setup/RootUserExists 检查后
// 各自尝试写入。修复前（无固定主键占位）会创建两个 root 用户和两条 setup
// 记录；修复后只允许一个成功，另一个按已初始化返回。
func TestInitializeSetupConcurrentInstancesCreateSingleRoot(t *testing.T) {
	setupSetupTestDB(t)

	const instances = 4
	var wg sync.WaitGroup
	errs := make([]error, instances)
	roots := make([]*User, instances)
	for i := 0; i < instances; i++ {
		roots[i] = newSetupRootUser(fmt.Sprintf("root-%d", i))
	}
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = InitializeSetup(roots[idx], NewSetupRecord("test", 1700000000))
		}(i)
	}
	wg.Wait()

	var successCount, alreadyInitialized int
	for _, err := range errs {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, ErrSetupAlreadyInitialized):
			alreadyInitialized++
		default:
			t.Fatalf("unexpected setup error: %v", err)
		}
	}
	if successCount != 1 {
		t.Fatalf("exactly one instance must initialize, got %d (alreadyInitialized=%d)", successCount, alreadyInitialized)
	}
	if alreadyInitialized != instances-1 {
		t.Fatalf("remaining instances must report already initialized, got %d", alreadyInitialized)
	}

	// 只有一个 root 用户
	var rootCount int64
	if err := DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Count(&rootCount).Error; err != nil {
		t.Fatalf("count root users: %v", err)
	}
	if rootCount != 1 {
		t.Fatalf("concurrent setup must create exactly one root user, got %d", rootCount)
	}

	// 只有一条 setup 记录
	var setupCount int64
	if err := DB.Model(&Setup{}).Count(&setupCount).Error; err != nil {
		t.Fatalf("count setup records: %v", err)
	}
	if setupCount != 1 {
		t.Fatalf("concurrent setup must create exactly one setup record, got %d", setupCount)
	}
}

// root 用户创建失败时整个事务回滚：不能留下"已初始化但无 root"的中间态。
func TestInitializeSetupRollsBackWhenRootCreationFails(t *testing.T) {
	setupSetupTestDB(t)

	// 预置同名用户，使 root 创建触发唯一约束失败
	existing := newSetupRootUser("dup-root")
	existing.Role = common.RoleCommonUser
	if err := DB.Create(existing).Error; err != nil {
		t.Fatalf("create conflicting user: %v", err)
	}

	root := newSetupRootUser("dup-root")
	err := InitializeSetup(root, NewSetupRecord("test", 1700000000))
	if err == nil {
		t.Fatal("expected setup to fail when root creation fails")
	}
	if errors.Is(err, ErrSetupAlreadyInitialized) {
		t.Fatal("root creation failure must not be reported as already initialized")
	}

	// setup 记录必须随事务回滚
	var setupCount int64
	if err := DB.Model(&Setup{}).Count(&setupCount).Error; err != nil {
		t.Fatalf("count setup records: %v", err)
	}
	if setupCount != 0 {
		t.Fatalf("setup record must roll back with root creation failure, got %d", setupCount)
	}
}

// root 已存在（历史状态）时，仅写 setup 记录的路径同样只允许一条。
func TestInitializeSetupWithoutRootIsIdempotentUnderConflict(t *testing.T) {
	setupSetupTestDB(t)

	if err := InitializeSetup(nil, NewSetupRecord("test", 1700000000)); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	err := InitializeSetup(nil, NewSetupRecord("test", 1700000001))
	if !errors.Is(err, ErrSetupAlreadyInitialized) {
		t.Fatalf("second setup must report already initialized, got %v", err)
	}

	var setupCount int64
	if err := DB.Model(&Setup{}).Count(&setupCount).Error; err != nil {
		t.Fatalf("count setup records: %v", err)
	}
	if setupCount != 1 {
		t.Fatalf("expected single setup record, got %d", setupCount)
	}
}
