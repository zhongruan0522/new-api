package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openDBByType(dsn string, dbType string) (*gorm.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("DSN 不能为空")
	}
	switch dbType {
	case common.DatabaseTypePostgreSQL:
		return gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			PrepareStmt: true,
		})
	case common.DatabaseTypeMySQL:
		dsn = ensureMySQLParseTime(dsn)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{
			PrepareStmt: true,
		})
	case common.DatabaseTypeSQLite:
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{
			PrepareStmt: true,
		})
	default:
		return nil, errors.New("不支持的数据库类型")
	}
}

func ensureMySQLParseTime(dsn string) string {
	if strings.Contains(dsn, "parseTime") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&parseTime=true"
	}
	return dsn + "?parseTime=true"
}

func closeGormDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
