package model

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

var commonGroupCol string
var commonKeyCol string
var commonTrueVal string
var commonFalseVal string

var logKeyCol string
var logGroupCol string

func init() {
	initCol()
}

func initCol() {
	// init common column names
	if common.UsingPostgreSQL {
		commonGroupCol = `"group"`
		commonKeyCol = `"key"`
		commonTrueVal = "true"
		commonFalseVal = "false"
	} else {
		commonGroupCol = "`group`"
		commonKeyCol = "`key`"
		commonTrueVal = "1"
		commonFalseVal = "0"
	}
	if os.Getenv("LOG_SQL_DSN") != "" {
		switch common.LogSqlType {
		case common.DatabaseTypePostgreSQL:
			logGroupCol = `"group"`
			logKeyCol = `"key"`
		default:
			logGroupCol = commonGroupCol
			logKeyCol = commonKeyCol
		}
	} else {
		// LOG_SQL_DSN 为空时，日志数据库与主数据库相同
		if common.UsingPostgreSQL {
			logGroupCol = `"group"`
			logKeyCol = `"key"`
		} else {
			logGroupCol = commonGroupCol
			logKeyCol = commonKeyCol
		}
	}
	// log sql type and database type
	//common.SysLog("Using Log SQL Type: " + common.LogSqlType)
}

var DB *gorm.DB

var LOG_DB *gorm.DB

const defaultSQLMaxIdleConns = 20

func createRootAccountIfNeed() error {
	var user User
	//if user.Status != common.UserStatusEnabled {
	if err := DB.First(&user).Error; err != nil {
		common.SysLog("no user exists, create a root user for you: username is root, password is 123456")
		hashedPassword, err := common.Password2Hash("123456")
		if err != nil {
			return err
		}
		rootUser := User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: nil,
			Quota:       100000000,
		}
		DB.Create(&rootUser)
	}
	return nil
}

func CheckSetup() {
	setup := GetSetup()
	if setup == nil {
		// No setup record exists, check if we have a root user
		if RootUserExists() {
			common.SysLog("system is not initialized, but root user exists")
			// Create setup record
			newSetup := Setup{
				Version:       common.Version,
				InitializedAt: time.Now().Unix(),
			}
			err := DB.Create(&newSetup).Error
			if err != nil {
				common.SysLog("failed to create setup record: " + err.Error())
			}
			constant.Setup = true
		} else {
			common.SysLog("system is not initialized and no root user exists")
			constant.Setup = false
		}
	} else {
		// Setup record exists, system is initialized
		common.SysLog("system is already initialized at: " + time.Unix(setup.InitializedAt, 0).String())
		constant.Setup = true
	}
}

func chooseDB(envName string, isLog bool) (*gorm.DB, error) {
	defer func() {
		initCol()
	}()
	dsn := os.Getenv(envName)
	if dsn != "" {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			// Use PostgreSQL
			common.SysLog("using PostgreSQL as database")
			if !isLog {
				common.UsingPostgreSQL = true
			} else {
				common.LogSqlType = common.DatabaseTypePostgreSQL
			}
			return gorm.Open(postgres.New(postgres.Config{
				DSN:                  dsn,
				PreferSimpleProtocol: true, // disables implicit prepared statement usage
			}), newGormConfig())
		}
		if strings.HasPrefix(dsn, "local") {
			common.SysLog("SQL_DSN not set, using SQLite as database")
			if !isLog {
				common.UsingSQLite = true
			} else {
				common.LogSqlType = common.DatabaseTypeSQLite
			}
			return gorm.Open(sqlite.Open(common.SQLitePath), newGormConfig())
		}
		// Use MySQL
		common.SysLog("using MySQL as database")
		// check parseTime
		if !strings.Contains(dsn, "parseTime") {
			if strings.Contains(dsn, "?") {
				dsn += "&parseTime=true"
			} else {
				dsn += "?parseTime=true"
			}
		}
		if !isLog {
			common.UsingMySQL = true
		} else {
			common.LogSqlType = common.DatabaseTypeMySQL
		}
		return gorm.Open(mysql.Open(dsn), newGormConfig())
	}
	// Use SQLite
	common.SysLog("SQL_DSN not set, using SQLite as database")
	common.UsingSQLite = true
	return gorm.Open(sqlite.Open(common.SQLitePath), newGormConfig())
}

// newGormConfig 构建统一的 GORM 配置。
//
// 关键点：默认 logger 的 IgnoreRecordNotFoundError 为 false，会把
// gorm.ErrRecordNotFound 当作 SQL 错误日志打印。在本项目中，按 key/id 查
// token、user、channel 等实体时"记录不存在"是合法的业务路径（无效 key、
// 过期、扫描、缓存回填探测等都会命中），不应该污染运行日志。
//
// 这里显式打开 IgnoreRecordNotFoundError，仅屏蔽 not-found 类日志，
// 慢查询和真正的 SQL 错误仍会正常输出。
func newGormConfig() *gorm.Config {
	logLevel := gormlogger.Warn
	if common.DebugEnabled {
		logLevel = gormlogger.Info
	}
	return &gorm.Config{
		// Prepared statement caching improves hot SQL paths but retains driver and
		// GORM statement state while idle. Keep it opt-in for small deployments.
		PrepareStmt: common.GetEnvOrDefaultBool("SQL_PREPARE_STMT", false),
		Logger: gormlogger.New(
			log.New(gin.DefaultWriter, "\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	}
}

func InitDB() (err error) {
	db, err := chooseDB("SQL_DSN", false)
	if err == nil {
		if common.DebugEnabled {
			db = db.Debug()
		}
		DB = db
		// MySQL charset/collation startup check: ensure Chinese-capable charset
		if common.UsingMySQL {
			if err := checkMySQLChineseSupport(DB); err != nil {
				panic(err)
			}
		}
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		sqlDB.SetMaxIdleConns(common.GetEnvOrDefault("SQL_MAX_IDLE_CONNS", defaultSQLMaxIdleConns))
		sqlDB.SetMaxOpenConns(common.GetEnvOrDefault("SQL_MAX_OPEN_CONNS", 1000))
		sqlDB.SetConnMaxLifetime(time.Second * time.Duration(common.GetEnvOrDefault("SQL_MAX_LIFETIME", 60)))

		if !common.IsMasterNode {
			return nil
		}
		if common.UsingMySQL {
			//_, _ = sqlDB.Exec("ALTER TABLE channels MODIFY model_mapping TEXT;") // TODO: delete this line when most users have upgraded
		}
		common.SysLog("database migration started")
		err = migrateDB()
		return err
	} else {
		common.FatalLog(err)
	}
	return err
}

func InitLogDB() (err error) {
	if os.Getenv("LOG_SQL_DSN") == "" {
		LOG_DB = DB
		return
	}
	db, err := chooseDB("LOG_SQL_DSN", true)
	if err == nil {
		if common.DebugEnabled {
			db = db.Debug()
		}
		LOG_DB = db
		// If log DB is MySQL, also ensure Chinese-capable charset
		if common.LogSqlType == common.DatabaseTypeMySQL {
			if err := checkMySQLChineseSupport(LOG_DB); err != nil {
				panic(err)
			}
		}
		sqlDB, err := LOG_DB.DB()
		if err != nil {
			return err
		}
		sqlDB.SetMaxIdleConns(common.GetEnvOrDefault("SQL_MAX_IDLE_CONNS", defaultSQLMaxIdleConns))
		sqlDB.SetMaxOpenConns(common.GetEnvOrDefault("SQL_MAX_OPEN_CONNS", 1000))
		sqlDB.SetConnMaxLifetime(time.Second * time.Duration(common.GetEnvOrDefault("SQL_MAX_LIFETIME", 60)))

		if !common.IsMasterNode {
			return nil
		}
		common.SysLog("database migration started")
		err = migrateLOGDB()
		return err
	} else {
		common.FatalLog(err)
	}
	return err
}

func migrateDB() error {
	// 清理旧版唯一约束/索引，防止 GORM AutoMigrate 的 MigrateColumnUnique 报 SQLSTATE 42704。
	// 详见 cleanupPrefillGroupLegacyIndex 和 CleanupLegacyUniqueConstraints 的注释。
	cleanupLegacyUniqueIndexes()

	// PostgreSQL：把旧库中可能漂移成 json/jsonb 的渠道 JSON-like 列改回 TEXT，
	// 避免写入空字符串/非 JSON 内容时触发 SQLSTATE 22P02。必须在 AutoMigrate 之前执行。
	cleanupLegacyChannelJSONColumns()

	// PostgreSQL：把旧版 tokens.model_limits 从 varchar(1024) 迁移为 text，
	// 避免超过 1024 字符的模型限制字符串写入失败。必须在 AutoMigrate 之前执行。
	if err := migrateTokenModelLimitsToText(); err != nil {
		return err
	}

	err := DB.AutoMigrate(
		&Channel{},
		&Ticket{},
		&TicketEntry{},
		&Token{},
		&User{},
		&PasskeyCredential{},
		&Option{},
		&Redemption{},
		&Ability{},
		&Log{},
		&StoredImage{},
		&StoredVideo{},
		&TopUp{},
		&QuotaData{},
		&Model{},
		&Vendor{},
		&PrefillGroup{},
		&Setup{},
		&TwoFA{},
		&TwoFABackupCode{},
		&Checkin{},
		&DynamicRatioRule{},
		&AuditLog{},
		&MiniMaxVoice{},
	)
	if err != nil {
		return err
	}

	if err := cleanupEmptyAccessTokens(); err != nil {
		return err
	}
	if err := cleanupRemovedChatPlaygroundData(); err != nil {
		return err
	}
	cleanupRemovedOAuth()
	cleanupRemovedTokenSetting()
	if err := cleanupRemovedMultimodalTextMode(); err != nil {
		return err
	}
	cleanupRemovedQuotaDataCacheStats()
	return nil
}

func migrateDBFast() error {
	// 同 migrateDB 中的说明
	cleanupLegacyUniqueIndexes()

	// PostgreSQL：把旧库中可能漂移成 json/jsonb 的渠道 JSON-like 列改回 TEXT，
	// 避免写入空字符串/非 JSON 内容时触发 SQLSTATE 22P02。必须在 AutoMigrate 之前执行。
	cleanupLegacyChannelJSONColumns()

	// PostgreSQL：把旧版 tokens.model_limits 从 varchar(1024) 迁移为 text，
	// 避免超过 1024 字符的模型限制字符串写入失败。必须在 AutoMigrate 之前执行。
	if err := migrateTokenModelLimitsToText(); err != nil {
		return err
	}

	var wg sync.WaitGroup

	migrations := []struct {
		model interface{}
		name  string
	}{
		{&Channel{}, "Channel"},
		{&Ticket{}, "Ticket"},
		{&TicketEntry{}, "TicketEntry"},
		{&Token{}, "Token"},
		{&User{}, "User"},
		{&PasskeyCredential{}, "PasskeyCredential"},
		{&Option{}, "Option"},
		{&Redemption{}, "Redemption"},
		{&Ability{}, "Ability"},
		{&Log{}, "Log"},
		{&StoredImage{}, "StoredImage"},
		{&StoredVideo{}, "StoredVideo"},
		{&TopUp{}, "TopUp"},
		{&QuotaData{}, "QuotaData"},
		{&Model{}, "Model"},
		{&Vendor{}, "Vendor"},
		{&PrefillGroup{}, "PrefillGroup"},
		{&Setup{}, "Setup"},
		{&TwoFA{}, "TwoFA"},
		{&TwoFABackupCode{}, "TwoFABackupCode"},
		{&Checkin{}, "Checkin"},
		{&DynamicRatioRule{}, "DynamicRatioRule"},
		{&AuditLog{}, "AuditLog"},
		{&MiniMaxVoice{}, "MiniMaxVoice"},
	}
	// 动态计算migration数量，确保errChan缓冲区足够大
	errChan := make(chan error, len(migrations))

	for _, m := range migrations {
		wg.Add(1)
		go func(model interface{}, name string) {
			defer wg.Done()
			if err := DB.AutoMigrate(model); err != nil {
				errChan <- fmt.Errorf("failed to migrate %s: %v", name, err)
			}
		}(m.model, m.name)
	}

	// Wait for all migrations to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	if err := cleanupEmptyAccessTokens(); err != nil {
		return err
	}
	if err := cleanupRemovedChatPlaygroundData(); err != nil {
		return err
	}
	cleanupRemovedOAuth()
	cleanupRemovedTokenSetting()
	if err := cleanupRemovedMultimodalTextMode(); err != nil {
		return err
	}
	cleanupRemovedQuotaDataCacheStats()
	common.SysLog("database migrated")
	return nil
}

func migrateLOGDB() error {
	var err error
	if err = LOG_DB.AutoMigrate(&Log{}); err != nil {
		return err
	}
	return nil
}

func cleanupEmptyAccessTokens() error {
	return DB.Model(&User{}).Where("access_token = ?", "").Update("access_token", nil).Error
}

// cleanupLegacyUniqueIndexes 清理所有从旧版 uniqueIndex tag 迁移到新版复合/部分索引后
// 可能残留的 UNIQUE 约束和索引。仅影响 PostgreSQL。
//
// 根因：GORM AutoMigrate 的 MigrateColumnUnique 会查询 information_schema.table_constraints，
// 如果检测到列有 UNIQUE 约束但模型定义中 field.Unique 为 false（因为用的是 uniqueIndex tag），
// 就会调用 DropConstraint（不带 IF EXISTS），导致 SQLSTATE 42704。
//
// 此函数在 AutoMigrate 之前执行，动态查询并删除所有相关的旧 UNIQUE 约束和索引。
func cleanupLegacyUniqueIndexes() {
	if !common.UsingPostgreSQL {
		return
	}
	// PrefillGroup: 旧版 gorm:"uniqueIndex" → 新版 gorm:"uniqueIndex:uk_prefill_name,where:deleted_at IS NULL"
	CleanupLegacyUniqueConstraints(DB, "prefill_groups", "name", []string{"uni_prefill_groups_name", "idx_prefill_groups_name"})
	// Model: 旧版 gorm:"uniqueIndex" → 新版 gorm:"uniqueIndex:uk_model_name_delete_at,priority:1"
	CleanupLegacyUniqueConstraints(DB, "models", "model_name", []string{"uni_models_model_name", "idx_models_model_name"})
	// Vendor: 旧版 gorm:"uniqueIndex" → 新版 gorm:"uniqueIndex:uk_vendor_name_delete_at,priority:1"
	CleanupLegacyUniqueConstraints(DB, "vendors", "name", []string{"uni_vendors_name", "idx_vendors_name"})
}

// cleanupLegacyChannelJSONColumns 把 PostgreSQL 旧库中可能漂移成 json/jsonb 的渠道
// JSON-like 列改回 TEXT。当前模型意图是把这些字段当作 JSON 文本存储（TEXT），
// 这样写入空字符串或非 JSON 内容不会被 PostgreSQL 校验拒绝。
//
// 仅影响 PostgreSQL；对 MySQL/SQLite 无操作。必须在 AutoMigrate(&Channel{}) 之前执行，
// 以免 AutoMigrate 检测到类型不一致时再触发列类型变更或报错。
//
// 幂等：只对实际类型为 json/jsonb 的列执行 ALTER，列已是 text 时跳过。
func cleanupLegacyChannelJSONColumns() {
	if !common.UsingPostgreSQL {
		return
	}
	// 当前模型中这些字段都是 TEXT 存储意图（gorm:"type:text" 或无 JSON 类型）
	columns := []string{"other", "setting", "param_override", "header_override", "settings"}
	for _, col := range columns {
		alterChannelColumnToTextIfJSON(DB, "channels", col)
	}
}

// alterChannelColumnToTextIfJSON 检查指定列的数据类型，若为 json/jsonb 则转为 text。
// 仅 PostgreSQL 调用。使用 information_schema 探测类型，避免硬编码 schema。
func alterChannelColumnToTextIfJSON(db *gorm.DB, tableName string, columnName string) {
	var dataType string
	err := db.Raw(
		`SELECT data_type FROM information_schema.columns WHERE table_name = ? AND column_name = ? LIMIT 1`,
		tableName, columnName,
	).Row().Scan(&dataType)
	if err != nil {
		// 表或列可能尚不存在（首次初始化），直接跳过，交给 AutoMigrate 创建。
		return
	}
	// data_type 为 'json' 或 'USER-DEFINED'（jsonb 在 information_schema 中显示为 USER-DEFINED，
	// 需进一步用 pg_attribute/format_type 精确判断）。这里同时处理两种情况。
	needsMigrate := false
	switch strings.ToLower(dataType) {
	case "json":
		needsMigrate = true
	case "user-defined":
		// 进一步用 pg_catalog.format_type 判断 jsonb
		var udtName string
		err = db.Raw(
			`SELECT format_type(a.atttypid, a.atttypmod) FROM pg_attribute a
			 JOIN pg_class c ON a.attrelid = c.oid
			 JOIN pg_namespace n ON c.relnamespace = n.oid
			 WHERE c.relname = ? AND a.attname = ? AND a.attnum > 0`,
			tableName, columnName,
		).Row().Scan(&udtName)
		if err == nil && (strings.Contains(strings.ToLower(udtName), "json")) {
			needsMigrate = true
		}
	}
	if !needsMigrate {
		return
	}
	// USING 子句把现有 JSON 值转换为 text；NULL 保持 NULL。
	// 表名/列名来自常量，不拼接外部输入，避免 SQL 注入。
	stmt := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s TYPE text USING %s::text`, tableName, columnName, columnName)
	if err := db.Exec(stmt).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to migrate channel column %s.%s from JSON to TEXT: %v", tableName, columnName, err))
	} else {
		common.SysLog(fmt.Sprintf("migrated channel column %s.%s from JSON to TEXT", tableName, columnName))
	}
}

// migrateTokenModelLimitsToText 把 PostgreSQL 旧库中的 tokens.model_limits 列
// 从 varchar(1024) 迁移为 text。旧版定义为 varchar(1024)，当模型限制字符串
// 超过 1024 字符时 PostgreSQL 会写入失败。
//
// 仅影响 PostgreSQL；对 MySQL/SQLite 无操作。必须在 AutoMigrate(&Token{}) 之前
// 执行，以免 AutoMigrate 检测到类型不一致时再触发列类型变更或报错。
//
// 幂等：列类型已经是 text 时跳过。
func migrateTokenModelLimitsToText() error {
	if !common.UsingPostgreSQL {
		return nil
	}
	var dataType string
	err := DB.Raw(
		`SELECT data_type FROM information_schema.columns WHERE table_name = ? AND column_name = ? LIMIT 1`,
		"tokens", "model_limits",
	).Row().Scan(&dataType)
	if err != nil {
		// 表或列可能尚不存在（首次初始化），直接跳过，交给 AutoMigrate 创建。
		return nil
	}
	if strings.EqualFold(dataType, "text") {
		return nil
	}
	// 表名/列名来自常量，不拼接外部输入，避免 SQL 注入。
	stmt := `ALTER TABLE tokens ALTER COLUMN model_limits TYPE text`
	if err := DB.Exec(stmt).Error; err != nil {
		return fmt.Errorf("failed to migrate tokens.model_limits to text: %w", err)
	}
	common.SysLog("migrated tokens.model_limits to text")
	return nil
}

// CleanupLegacyUniqueConstraints 动态查询并删除指定表/列的所有 UNIQUE 约束和已知旧索引。
// 用于解决 GORM AutoMigrate 的 MigrateColumnUnique 在约束不存在时 DROP CONSTRAINT 不带 IF EXISTS 的问题。
func CleanupLegacyUniqueConstraints(db *gorm.DB, tableName string, columnName string, legacyIndexNames []string) {
	// 1. 动态查询所有与指定列相关的 UNIQUE 约束，逐一删除
	rows, err := db.Raw(
		`SELECT tc.constraint_name FROM information_schema.table_constraints tc
		 JOIN information_schema.constraint_column_usage ccu ON tc.constraint_schema = ccu.constraint_schema AND tc.constraint_name = ccu.constraint_name
		 WHERE tc.constraint_type = 'UNIQUE' AND tc.table_name = ? AND ccu.column_name = ?`,
		tableName, columnName,
	).Rows()
	if err == nil {
		for rows.Next() {
			var constraintName string
			if rows.Scan(&constraintName) == nil {
				_ = db.Exec(`ALTER TABLE ? DROP CONSTRAINT IF EXISTS ?`, clause.Table{Name: tableName}, clause.Column{Name: constraintName}).Error
			}
		}
		rows.Close()
	}

	// 2. 清理已知旧索引名
	for _, idxName := range legacyIndexNames {
		_ = db.Exec(`DROP INDEX IF EXISTS ?`, clause.Column{Name: idxName}).Error
	}
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	return err
}

func CloseDB() error {
	if LOG_DB != DB {
		err := closeDB(LOG_DB)
		if err != nil {
			return err
		}
	}
	return closeDB(DB)
}

// checkMySQLChineseSupport ensures the MySQL connection and current schema
// default charset/collation can store Chinese characters. It allows common
// Chinese-capable charsets (utf8mb4, utf8, gbk, big5, gb18030) and panics otherwise.
func checkMySQLChineseSupport(db *gorm.DB) error {
	// 仅检测：当前库默认字符集/排序规则 + 各表的排序规则（隐含字符集）

	// Read current schema defaults
	var schemaCharset, schemaCollation string
	err := db.Raw("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = DATABASE()").Row().Scan(&schemaCharset, &schemaCollation)
	if err != nil {
		return fmt.Errorf("读取当前库默认字符集/排序规则失败 / Failed to read schema default charset/collation: %v", err)
	}

	toLower := func(s string) string { return strings.ToLower(s) }
	// Allowed charsets that can store Chinese text
	allowedCharsets := map[string]string{
		"utf8mb4": "utf8mb4_",
		"utf8":    "utf8_",
		"gbk":     "gbk_",
		"big5":    "big5_",
		"gb18030": "gb18030_",
	}
	isChineseCapable := func(cs, cl string) bool {
		csLower := toLower(cs)
		clLower := toLower(cl)
		if prefix, ok := allowedCharsets[csLower]; ok {
			if clLower == "" {
				return true
			}
			return strings.HasPrefix(clLower, prefix)
		}
		// 如果仅提供了排序规则，尝试按排序规则前缀判断
		for _, prefix := range allowedCharsets {
			if strings.HasPrefix(clLower, prefix) {
				return true
			}
		}
		return false
	}

	// 1) 当前库默认值必须支持中文
	if !isChineseCapable(schemaCharset, schemaCollation) {
		return fmt.Errorf("当前库默认字符集/排序规则不支持中文：schema(%s/%s)。请将库设置为 utf8mb4/utf8/gbk/big5/gb18030 / Schema default charset/collation is not Chinese-capable: schema(%s/%s). Please set to utf8mb4/utf8/gbk/big5/gb18030",
			schemaCharset, schemaCollation, schemaCharset, schemaCollation)
	}

	// 2) 所有物理表的排序规则（隐含字符集）必须支持中文
	type tableInfo struct {
		Name      string
		Collation *string
	}
	var tables []tableInfo
	if err := db.Raw("SELECT TABLE_NAME, TABLE_COLLATION FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'").Scan(&tables).Error; err != nil {
		return fmt.Errorf("读取表排序规则失败 / Failed to read table collations: %v", err)
	}

	var badTables []string
	for _, t := range tables {
		// NULL 或空表示继承库默认设置，已在上面校验库默认，视为通过
		if t.Collation == nil || *t.Collation == "" {
			continue
		}
		cl := *t.Collation
		// 仅凭排序规则判断是否中文可用
		ok := false
		lower := strings.ToLower(cl)
		for _, prefix := range allowedCharsets {
			if strings.HasPrefix(lower, prefix) {
				ok = true
				break
			}
		}
		if !ok {
			badTables = append(badTables, fmt.Sprintf("%s(%s)", t.Name, cl))
		}
	}

	if len(badTables) > 0 {
		// 限制输出数量以避免日志过长
		maxShow := 20
		shown := badTables
		if len(shown) > maxShow {
			shown = shown[:maxShow]
		}
		return fmt.Errorf(
			"存在不支持中文的表，请修复其排序规则/字符集。示例（最多展示 %d 项）：%v / Found tables not Chinese-capable. Please fix their collation/charset. Examples (showing up to %d): %v",
			maxShow, shown, maxShow, shown,
		)
	}
	return nil
}

var (
	lastPingTime time.Time
	pingMutex    sync.Mutex
)

func PingDB() error {
	pingMutex.Lock()
	defer pingMutex.Unlock()

	if time.Since(lastPingTime) < time.Second*10 {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Error getting sql.DB from GORM: %v", err)
		return err
	}

	err = sqlDB.Ping()
	if err != nil {
		log.Printf("Error pinging DB: %v", err)
		return err
	}

	lastPingTime = time.Now()
	common.SysLog("Database pinged successfully")
	return nil
}
