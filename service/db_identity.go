package service

import (
	"database/sql"
	"fmt"

	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

// dbIdentity 用于唯一标识一个数据库连接
type dbIdentity struct {
	Host       string
	Port       string
	Database   string
	ServerUUID string // MySQL: @@server_uuid; PostgreSQL: system_identifier
}

func (id dbIdentity) String() string {
	return fmt.Sprintf("%s:%s/%s (uuid=%s)", id.Host, id.Port, id.Database, id.ServerUUID)
}

// getDBIdentity 通过数据库连接获取唯一标识
func getDBIdentity(db *gorm.DB, dbType string) (dbIdentity, error) {
	switch dbType {
	case common.DatabaseTypePostgreSQL:
		return getPostgresIdentity(db)
	case common.DatabaseTypeMySQL:
		return getMySQLIdentity(db)
	default:
		return dbIdentity{}, fmt.Errorf("不支持的数据库类型：%s", dbType)
	}
}

func getPostgresIdentity(db *gorm.DB) (dbIdentity, error) {
	var host, port, dbName string
	var sysID string
	// pg 后端进程监听地址 + system_identifier（唯一标识一个 PG 实例）
	row := db.Raw("SELECT inet_server_addr(), inet_server_port(), current_database(), system_identifier FROM pg_control_system()").Row()
	if err := row.Scan(&host, &port, &dbName, &sysID); err != nil {
		return dbIdentity{}, fmt.Errorf("获取 PostgreSQL 连接标识失败：%w", err)
	}
	return dbIdentity{Host: host, Port: port, Database: dbName, ServerUUID: sysID}, nil
}

func getMySQLIdentity(db *gorm.DB) (dbIdentity, error) {
	var host, port sql.NullString
	var dbName, serverUUID string
	// @@server_uuid 全局唯一标识一个 MySQL 实例（5.6+）
	row := db.Raw("SELECT @@hostname, @@port, DATABASE(), @@server_uuid").Row()
	if err := row.Scan(&host, &port, &dbName, &serverUUID); err != nil {
		return dbIdentity{}, fmt.Errorf("获取 MySQL 连接标识失败：%w", err)
	}
	h := ""
	if host.Valid {
		h = host.String
	}
	p := ""
	if port.Valid {
		p = port.String
	}
	return dbIdentity{Host: h, Port: p, Database: dbName, ServerUUID: serverUUID}, nil
}

// isSameDBConnection 判断两个数据库连接是否指向同一个数据库
func isSameDBConnection(src *gorm.DB, dst *gorm.DB, dbType string) (bool, error) {
	srcID, err := getDBIdentity(src, dbType)
	if err != nil {
		return false, fmt.Errorf("获取源库标识失败：%w", err)
	}
	dstID, err := getDBIdentity(dst, dbType)
	if err != nil {
		return false, fmt.Errorf("获取目标库标识失败：%w", err)
	}
	return srcID == dstID, nil
}
