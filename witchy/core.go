// @author 2021年06月06日22:21:57
// @author guzzsek
// @desc 加入日志追踪, 日志注入 trace id

package witchy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"sync"
	"time"
)

// Colors
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

var tracer = otel.Tracer("github.com/guzzsek/gorm_log")

func NewPaper(logger *zap.Logger, config gormLogger.Config) gormLogger.Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = Green + "%s\n" + Reset + Green + "[info] " + Reset
		warnStr = BlueBold + "%s\n" + Reset + Magenta + "[warn] " + Reset
		errStr = Magenta + "%s\n" + Reset + Red + "[error] " + Reset
		traceStr = Green + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
		traceWarnStr = Green + "%s " + Yellow + "%s\n" + Reset + RedBold + "[%.3fms] " + Yellow + "[rows:%v]" + Magenta + " %s" + Reset
		traceErrStr = RedBold + "%s " + MagentaBold + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
	}

	return &Paper{
		Config:       config,
		logger:       logger,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

type Paper struct {
	gormLogger.Config

	logger *zap.Logger
	pool   *sync.Pool

	infoStr string
	warnStr string
	errStr  string

	traceStr     string
	traceErrStr  string
	traceWarnStr string
}

func (c *Paper) middleEntry(field ...zap.Field) *middleEntry {
	return &middleEntry{
		logger: c.logger,
		pool:   c.pool,
		field:  field,
	}
}

type middleEntry struct {
	logger *zap.Logger
	pool   *sync.Pool
	field  []zap.Field
}

func (c *middleEntry) Printf(msg string, args ...interface{}) {
	buff := c.pool.Get().(*bytes.Buffer)
	fmt.Fprintf(buff, msg, args...)
	c.logger.Info(buff.String(), c.field...)
	buff.Reset()
	c.pool.Put(buff)
}

func (c *Paper) Info(ctx context.Context, msg string, args ...interface{}) {
	if c.LogLevel < gormLogger.Info {
		return
	}

	c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.infoStr+msg, append([]interface{}{utils.FileWithLineNum()}, args...)...)
}

func (c *Paper) Warn(ctx context.Context, msg string, args ...interface{}) {
	if c.LogLevel < gormLogger.Warn {
		return
	}

	c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, args...)...)

}

func (c *Paper) Error(ctx context.Context, msg string, args ...interface{}) {
	if c.LogLevel < gormLogger.Error {
		return
	}

	c.middleEntry(zap.String("trace_id", traceID(ctx))).
		Printf(c.errStr+msg, append([]interface{}{utils.FileWithLineNum()}, args...)...)

}

func (c *Paper) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if c.LogLevel <= gormLogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	if trace.SpanFromContext(ctx).IsRecording() {
		sql, cnt := fc()
		_, span := tracer.Start(ctx, "trace")
		defer func() {
			span.End()
		}()
		span.SetAttributes(
			attribute.String("db.system", "gorm"),
			attribute.String("db.sql", sql),
			attribute.Int64("exec.affected", cnt),
			attribute.Int64("exec.latency", elapsed.Nanoseconds()/10e6),
		)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}

	switch {
	case err != nil && c.LogLevel >= gormLogger.Error && (!errors.Is(err, gormLogger.ErrRecordNotFound) || !c.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > c.SlowThreshold && c.SlowThreshold != 0 && c.LogLevel >= gormLogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", c.SlowThreshold)
		if rows == -1 {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case c.LogLevel == gormLogger.Info:
		sql, rows := fc()
		if rows == -1 {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			c.middleEntry(zap.String("trace_id", traceID(ctx))).Printf(c.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

func traceID(ctx context.Context) string {
	var traceID string
	if tid := trace.SpanContextFromContext(ctx).TraceID(); tid.IsValid() {
		traceID = tid.String()
	}
	return traceID
}
func (c *Paper) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	newlogger := *c
	newlogger.LogLevel = level
	return &newlogger
}
