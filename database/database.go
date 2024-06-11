package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/kc-whjf/alpha/aconfig"
	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/alog/gormwrapper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func MustNewDBWith(portName aconfig.PortName, appConfig *aconfig.Application, driver string, commonConfig *aconfig.Database) *gorm.DB {
	db, err := NewDBWith(portName, appConfig, driver, commonConfig)
	if err != nil {
		panic(err)
	}

	return db
}

func NewDBWith(portName aconfig.PortName, appConfig *aconfig.Application, driver string, commonConfig *aconfig.Database) (*gorm.DB, error) {
	location := appConfig.GetMatchedPrimaryPortLocation(portName)
	if location == nil {
		return nil, fmt.Errorf("missing matched primaryport location(%s)", portName)
	}
	options := appConfig.GetSecondaryPort(portName).Options
	if options == nil {
		return nil, fmt.Errorf("missing options for secondary port (%s)", portName)
	}

	return NewDB(driver, formatDNS(location, options), commonConfig)
}

func NewDB(driver, dsn string, commonConfig *aconfig.Database) (*gorm.DB, error) {
	alog.Sugar.Infof("database.NewDB: driver(%s) dsn(%s)", driver, dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormwrapper.New(alog.Sugar, gormwrapper.Config{
		SlowThreshold: time.Duration(commonConfig.SlowThresholdMilliseconds) * time.Millisecond,
		LogLevel:      gormlogger.Info,
	})})

	if err != nil {
		alog.Sugar.Errorf("database.NewDB failed: %v", err)
		return nil, err
	}

	stdDB, err := db.DB()
	if err != nil {
		alog.Sugar.Errorf("database.NewDB get standard DB failed: %v", err)
		return nil, err
	}
	stdDB.SetMaxOpenConns(commonConfig.MaxOpenConnections)
	stdDB.SetMaxIdleConns(commonConfig.MaxIdleConnections)
	stdDB.SetConnMaxLifetime(time.Duration(commonConfig.ConnectionMaxLifeSeconds) * time.Second)
	stdDB.SetConnMaxIdleTime(time.Duration(commonConfig.ConnectionMaxIdleSeconds) * time.Second)

	return db, nil
}

func formatDNS(location *aconfig.Location, options aconfig.KV) string {
	user := options.GetString("user")
	password := options.GetString("password")
	database := options.GetString("database")

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True", user, password, location.Address, location.Port, database)
}

func IsRecordNotfound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
