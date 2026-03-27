package service

import (
	"database/sql"
	"fmt"

	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

// dbIdentity 用于唯一标识一个数据库连接
type dbIdentity struct {
	Host     string
	Port     string
	Database string
}

// getDBIdentity 通过数据库连接获取 host/port/database 标识
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
	// pg 后端进程监听地址
	row := db.Raw("SELECT inet_server_addr(), inet_server_port(), current_database()").Row()
	if err := row.Scan(&host, &port, &dbName); err != nil {
		return dbIdentity{}, fmt.Errorf("获取 PostgreSQL 连接标识失败：%w", err)
	}
	return dbIdentity{Host: host, Port: port, Database: dbName}, nil
}

func getMySQLIdentity(db *gorm.DB) (dbIdentity, error) {
	// MySQL 没有直接的 inet_server_addr，用 @@hostname + @@port + DATABASE()
	var host, port sql.NullString
	var dbName string
	row := db.Raw("SELECT @@hostname, @@port, DATABASE()").Row()
	if err := row.Scan(&host, &port, &dbName); err != nil {
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
	return dbIdentity{Host: h, Port: p, Database: dbName}, nil
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
