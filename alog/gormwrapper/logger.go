package gormwrapper

import (
	"context"
	"fmt"
	"time"

	"github.com/kc-whjf/alpha/alog"
	"github.com/kc-whjf/alpha/autil/ahttp/request"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type Config struct {
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}

func New(sugarLogger *zap.SugaredLogger, config Config) gormlogger.Interface {
	var (
		infoStr      = "%s "
		warnStr      = "%s "
		errStr       = "%s "
		traceStr     = "%s [%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s [%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s [%.3fms] [rows:%v] %s"
	)

	return &logger{
		sugarLogger:  sugarLogger,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

type logger struct {
	sugarLogger *zap.SugaredLogger
	Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// LogMode log mode
func (l *logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.sugarLogger.Infow(
			fmt.Sprintf(l.infoStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...),
			alog.RequestIdKey,
			request.RequestIdValue(ctx))
	}
}

// Warn print warn messages
func (l logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.sugarLogger.Warnw(
			fmt.Sprintf(l.warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...),
			alog.RequestIdKey,
			request.RequestIdValue(ctx))
	}
}

// Error print error messages
func (l logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.sugarLogger.Errorw(
			fmt.Sprintf(l.errStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...),
			alog.RequestIdKey,
			request.RequestIdValue(ctx))
	}
}

// Trace print sql message
func (l logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel > 0 {
		elapsed := time.Since(begin)
		switch {
		case err != nil && l.LogLevel >= gormlogger.Error:
			sql, rows := fc()
			if rows == -1 {
				l.sugarLogger.Errorw(
					fmt.Sprintf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			} else {
				l.sugarLogger.Errorw(
					fmt.Sprintf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			}
		case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
			sql, rows := fc()
			slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
			if rows == -1 {
				l.sugarLogger.Warnw(
					fmt.Sprintf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			} else {
				l.sugarLogger.Warnw(
					fmt.Sprintf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			}
		case l.LogLevel >= gormlogger.Info:
			sql, rows := fc()
			if rows == -1 {
				l.sugarLogger.Infow(
					fmt.Sprintf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			} else {
				l.sugarLogger.Infow(
					fmt.Sprintf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql),
					alog.RequestIdKey,
					request.RequestIdValue(ctx))
			}
		}
	}
}
