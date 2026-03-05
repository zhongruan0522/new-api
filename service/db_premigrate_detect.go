package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func currentMainDBType() (string, error) {
	switch {
	case common.UsingPostgreSQL:
		return common.DatabaseTypePostgreSQL, nil
	case common.UsingMySQL:
		return common.DatabaseTypeMySQL, nil
	case common.UsingSQLite:
		return common.DatabaseTypeSQLite, nil
	default:
		return "", errors.New("无法识别当前主数据库类型（UsingMySQL/UsingPostgreSQL/UsingSQLite 均为 false）")
	}
}

func currentLogDBType() (string, error) {
	// LOG_SQL_DSN 为空时，日志库与主库相同；否则用 common.LogSqlType
	mainType, err := currentMainDBType()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(common.GetEnvOrDefaultString("LOG_SQL_DSN", "")) == "" {
		return mainType, nil
	}
	switch common.LogSqlType {
	case common.DatabaseTypePostgreSQL, common.DatabaseTypeMySQL, common.DatabaseTypeSQLite:
		return common.LogSqlType, nil
	default:
		return "", errors.New("无法识别当前日志数据库类型")
	}
}

func parseDBTypeFromDSN(dsn string) (string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", errors.New("DSN 不能为空")
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return common.DatabaseTypePostgreSQL, nil
	}
	if strings.HasPrefix(dsn, "local") {
		return common.DatabaseTypeSQLite, nil
	}
	return common.DatabaseTypeMySQL, nil
}

func validateDBPreMigrateDirection(sourceType string, targetType string) error {
	if sourceType == "" || targetType == "" {
		return errors.New("数据库类型不能为空")
	}
	if sourceType == targetType {
		return errors.New("源数据库与目标数据库类型相同，无需迁移")
	}
	switch sourceType {
	case common.DatabaseTypeMySQL:
		if targetType != common.DatabaseTypePostgreSQL {
			return errors.New("仅支持 MySQL -> PostgreSQL 的预迁移")
		}
	case common.DatabaseTypePostgreSQL:
		if targetType != common.DatabaseTypeMySQL {
			return errors.New("仅支持 PostgreSQL -> MySQL 的预迁移")
		}
	case common.DatabaseTypeSQLite:
		if targetType != common.DatabaseTypePostgreSQL && targetType != common.DatabaseTypeMySQL {
			return errors.New("仅支持 SQLite -> PostgreSQL / SQLite -> MySQL 的预迁移")
		}
	default:
		return errors.New("不支持的源数据库类型")
	}
	return nil
}
