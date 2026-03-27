package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"

	"gorm.io/gorm"
)

func runDBSameTypeMigrateJob(job *DBSameTypeMigrateJob, params DBSameTypeMigrateStartParams) {
	ctx := context.Background()
	if err := runDBSameTypeMigrateJobWithContext(ctx, job, params); err != nil {
		job.markFailed(err)
		return
	}
	job.markSuccess()
}

func runDBSameTypeMigrateJobWithContext(ctx context.Context, job *DBSameTypeMigrateJob, params DBSameTypeMigrateStartParams) error {
	if job == nil {
		return errors.New("job 不能为空")
	}

	targetType, err := parseDBTypeFromDSN(params.TargetDSN)
	if err != nil {
		return err
	}

	job.appendLog(fmt.Sprintf("[%s] 同类型迁移启动：%s -> %s", time.Now().Format(time.RFC3339), job.SourceDBType, targetType))
	job.appendLog(fmt.Sprintf("[%s] 迁移日志：%v；覆盖目标库：%v", time.Now().Format(time.RFC3339), params.IncludeLogs, params.Force))

	// ============ 第一阶段：打开并校验所有目标库（P2: 先校验再写数据） ============

	job.setStep("校验目标数据库连接")

	// 准备目标主库：连接 + 建表 + 空库校验 + 身份校验
	targetMainDB, err := prepareSameTypeTargetMainDB(ctx, job, params, targetType)
	if err != nil {
		return err
	}
	defer func() { _ = closeGormDB(targetMainDB) }()

	// 校验目标主库 != 源主库（P1: 连接级标识，不依赖 DSN 字符串）
	if err := checkNotSameDB(job, model.DB, targetMainDB, targetType, "主库"); err != nil {
		return err
	}

	// 准备目标日志库（可选）：连接 + 建表 + 空库校验 + 身份校验
	// 在主库表复制之前完成，避免主库已写入但日志库校验失败导致半完成状态
	targetLogDB, needCloseLogDB, err := prepareSameTypeTargetLogDB(ctx, job, params, targetMainDB, targetType)
	if err != nil {
		return err
	}
	if needCloseLogDB {
		defer func() { _ = closeGormDB(targetLogDB) }()
	}

	// 校验目标日志库 != 源日志库
	if params.IncludeLogs {
		if err := checkNotSameDB(job, model.LOG_DB, targetLogDB, targetType, "日志库"); err != nil {
			return err
		}
	}

	// ============ 第二阶段：所有校验通过，开始复制数据 ============

	if err := migrateSameTypeMainDBTables(ctx, job, model.DB, targetMainDB, params); err != nil {
		return err
	}

	if params.IncludeLogs {
		if err := migrateSameTypeLogTable(ctx, job, model.LOG_DB, targetLogDB, params); err != nil {
			return err
		}
	}

	// PostgreSQL 序列修复
	if err := fixSameTypeTargetAutoIncrement(job, targetType, targetMainDB, targetLogDB, needCloseLogDB, params.IncludeLogs); err != nil {
		return err
	}

	job.setStep("完成")
	job.appendLog(fmt.Sprintf("[%s] 同类型迁移完成", time.Now().Format(time.RFC3339)))
	return nil
}

// checkNotSameDB 通过连接级数据库标识比较两个连接是否指向同一个库
func checkNotSameDB(job *DBSameTypeMigrateJob, src *gorm.DB, dst *gorm.DB, dbType string, label string) error {
	same, err := isSameDBConnection(src, dst, dbType)
	if err != nil {
		return fmt.Errorf("无法校验目标%s与源%s是否相同：%w（身份校验失败时拒绝继续，以防止误操作）", label, label, err)
	}
	if same {
		return fmt.Errorf("目标%s与当前源%s是同一个数据库，禁止迁移以避免数据被覆盖", label, label)
	}
	job.appendLog(fmt.Sprintf("[%s] 目标%s与源%s不是同一个数据库，校验通过", time.Now().Format(time.RFC3339), label, label))
	return nil
}

func prepareSameTypeTargetMainDB(ctx context.Context, job *DBSameTypeMigrateJob, params DBSameTypeMigrateStartParams, targetType string) (*gorm.DB, error) {
	job.appendLog(fmt.Sprintf("[%s] 连接目标主库...", time.Now().Format(time.RFC3339)))
	targetMainDB, err := openDBByType(params.TargetDSN, targetType)
	if err != nil {
		return nil, fmt.Errorf("连接目标主库失败：%w", err)
	}
	job.appendLog(fmt.Sprintf("[%s] 目标主库建表/迁移 schema...", time.Now().Format(time.RFC3339)))
	if err := autoMigrateTargetMainSchema(targetMainDB); err != nil {
		_ = closeGormDB(targetMainDB)
		return nil, fmt.Errorf("目标主库建表/迁移失败：%w", err)
	}
	if err := ensureSameTypeTargetMainDBEmptyOrForce(ctx, targetMainDB, params.Force); err != nil {
		_ = closeGormDB(targetMainDB)
		return nil, err
	}
	job.appendLog(fmt.Sprintf("[%s] 目标主库准备完成", time.Now().Format(time.RFC3339)))
	return targetMainDB, nil
}

func ensureSameTypeTargetMainDBEmptyOrForce(ctx context.Context, target *gorm.DB, force bool) error {
	if force {
		return nil
	}
	checkModels := []any{
		&model.User{},
		&model.Option{},
		&model.Channel{},
		&model.Token{},
	}
	for _, m := range checkModels {
		has, err := hasAnyRow(ctx, target, m)
		if err != nil {
			return fmt.Errorf("检查目标主库是否为空失败：%w", err)
		}
		if has {
			return errors.New("目标主库不是空库：请使用全新的空数据库，或勾选「覆盖目标库（清空后迁移）」")
		}
	}
	return nil
}

func prepareSameTypeTargetLogDB(ctx context.Context, job *DBSameTypeMigrateJob, params DBSameTypeMigrateStartParams, targetMainDB *gorm.DB, targetType string) (*gorm.DB, bool, error) {
	if !params.IncludeLogs {
		return targetMainDB, false, nil
	}

	targetLogDSN := strings.TrimSpace(params.TargetLogDSN)
	if targetLogDSN == "" {
		targetLogDSN = strings.TrimSpace(params.TargetDSN)
	}
	logType, err := parseDBTypeFromDSN(targetLogDSN)
	if err != nil {
		return nil, false, err
	}
	if logType != targetType {
		return nil, false, errors.New("目标日志库类型必须与目标主库一致")
	}
	if strings.TrimSpace(targetLogDSN) == strings.TrimSpace(params.TargetDSN) {
		return targetMainDB, false, nil
	}

	job.appendLog(fmt.Sprintf("[%s] 连接独立的目标日志库...", time.Now().Format(time.RFC3339)))
	targetLogDB, err := openDBByType(targetLogDSN, logType)
	if err != nil {
		return nil, false, fmt.Errorf("连接目标日志库失败：%w", err)
	}
	if err := autoMigrateTargetLogSchema(targetLogDB); err != nil {
		_ = closeGormDB(targetLogDB)
		return nil, false, fmt.Errorf("目标日志库建表/迁移失败：%w", err)
	}
	if !params.Force {
		has, err := hasAnyRow(ctx, targetLogDB, &model.Log{})
		if err != nil {
			_ = closeGormDB(targetLogDB)
			return nil, false, fmt.Errorf("检查目标日志库是否为空失败：%w", err)
		}
		if has {
			_ = closeGormDB(targetLogDB)
			return nil, false, errors.New("目标日志库不是空库：请使用全新的空数据库，或勾选「覆盖目标库（清空后迁移）」")
		}
	}
	job.appendLog(fmt.Sprintf("[%s] 目标日志库准备完成", time.Now().Format(time.RFC3339)))
	return targetLogDB, true, nil
}

func migrateSameTypeMainDBTables(ctx context.Context, job *DBSameTypeMigrateJob, src *gorm.DB, dst *gorm.DB, params DBSameTypeMigrateStartParams) error {
	for _, step := range dbPreMigrateMainSteps {
		if err := runSameTypeCopyStep(ctx, job, step, src, dst, params); err != nil {
			return err
		}
	}
	return nil
}

func migrateSameTypeLogTable(ctx context.Context, job *DBSameTypeMigrateJob, src *gorm.DB, dst *gorm.DB, params DBSameTypeMigrateStartParams) error {
	if src == nil {
		return errors.New("源日志库连接为空")
	}
	if dst == nil {
		return errors.New("目标日志库连接为空")
	}
	return runSameTypeCopyStep(ctx, job, dbPreMigrateLogStep, src, dst, params)
}

// runSameTypeCopyStep 通过 dbPreMigrateStep 接口执行表复制，并将进度回调到同类型迁移的 job
func runSameTypeCopyStep(ctx context.Context, job *DBSameTypeMigrateJob, step dbPreMigrateStep, src *gorm.DB, dst *gorm.DB, params DBSameTypeMigrateStartParams) error {
	job.setStep("迁移表：" + step.Name())
	job.appendLog(fmt.Sprintf("[%s] 开始迁移表 %s", time.Now().Format(time.RFC3339), step.Name()))

	// 使用一个临时的 DBPreMigrateJob 来接收 step.Run 的进度回调
	tmpJob := newDBPreMigrateJob("", job.SourceDBType, job.TargetDBType, DBPreMigrateStartParams{Force: params.Force})
	if err := step.Run(ctx, tmpJob, src, dst, DBPreMigrateStartParams{Force: params.Force}); err != nil {
		job.appendLog(fmt.Sprintf("[%s] 迁移表 %s 失败：%v", time.Now().Format(time.RFC3339), step.Name(), err))
		return err
	}

	// 将临时 job 的进度同步到同类型迁移的 job
	snap := tmpJob.snapshot()
	for _, t := range snap.Tables {
		job.setTableProgress(t.Name, t.Copied, t.Total)
	}
	job.appendLog(fmt.Sprintf("[%s] 迁移表 %s 完成", time.Now().Format(time.RFC3339), step.Name()))
	return nil
}

func fixSameTypeTargetAutoIncrement(job *DBSameTypeMigrateJob, targetType string, mainDB *gorm.DB, logDB *gorm.DB, logDBNeedClose bool, includeLogs bool) error {
	if targetType != common.DatabaseTypePostgreSQL {
		return nil
	}

	job.setStep("修复 PostgreSQL 自增序列")
	updated, err := resetPostgresOwnedSequences(mainDB)
	if err != nil {
		return fmt.Errorf("修复 PostgreSQL 自增序列失败：%w", err)
	}
	job.appendLog(fmt.Sprintf("[%s] PostgreSQL 自增序列已修复：%d 个", time.Now().Format(time.RFC3339), updated))

	if includeLogs && logDBNeedClose {
		updated, err := resetPostgresOwnedSequences(logDB)
		if err != nil {
			return fmt.Errorf("修复 PostgreSQL(日志库) 自增序列失败：%w", err)
		}
		job.appendLog(fmt.Sprintf("[%s] PostgreSQL(日志库) 自增序列已修复：%d 个", time.Now().Format(time.RFC3339), updated))
	}
	return nil
}
