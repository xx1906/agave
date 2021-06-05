package pencil

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/guzzsek/agave/pencil/config"
	"github.com/natefinch/lumberjack"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	_defaultBackUp = 200  // 保留日志的最大值
	_defaultSize   = 1024 // 默认日志最大分割容量
	_defaultAge    = 7    // 日志保留的最大天数
)

type Core struct {
	logger *zap.Logger
	pool   *sync.Pool
}

func NewCore(cfg *config.Config) (c *Core) {
	if cfg == nil {
		panic(fmt.Sprintf("NewCore cfg could be nil"))
	}

	var err error
	if err = os.MkdirAll(cfg.Path, 777); err != nil {
		panic(err)
	}
	var logLevel = zap.DebugLevel

	switch cfg.Level {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	default:
		logLevel = zap.InfoLevel
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "ts",
		CallerKey:     "file",
		EncodeLevel:   zapcore.CapitalLevelEncoder, // 大写 level 提示
		EncodeCaller:  zapcore.ShortCallerEncoder,  // 段文件 比如: core_log/NewCore.go:62
		StacktraceKey: "stack",

		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05:06"))
		}, //输出的时间格式
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 10e6)
		}, //
	}

	// debug  level log, 如果是 debug 级别, 将会输出所有的日志
	debugLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.DebugLevel && lvl >= logLevel
	})
	// info level log
	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.InfoLevel && lvl >= logLevel
	})

	// warn level log
	warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.WarnLevel && lvl >= logLevel
	})

	// error level log
	errorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.ErrorLevel && lvl >= logLevel
	})

	// 保留文件的最大数量
	var maxBackupSize = _defaultBackUp
	if cfg.MaxBackup != nil {
		maxBackupSize = cast.ToInt(cfg.GetMaxBackup())
	}

	// 保留日志的最大天数
	var maxAge = _defaultAge
	if cfg.MaxAge != nil {
		maxAge = cast.ToInt(cfg.GetMaxAge())
	}

	// 日志的最大值
	var maxSize = _defaultSize
	if cfg.MaxSize != nil {
		maxSize = cast.ToInt(cfg.GetMaxSize())
	}

	// writer
	debugWriter := getWriter(cfg.Path+string(filepath.Separator)+"debug.log", maxBackupSize, maxAge, maxSize, cfg.GetCompress())
	infoWriter := getWriter(cfg.Path+string(filepath.Separator)+"info.log", maxBackupSize, maxAge, maxSize, cfg.GetCompress())
	warnWriter := getWriter(cfg.Path+string(filepath.Separator)+"warn.log", maxBackupSize, maxAge, maxSize, cfg.GetCompress())
	errorWriter := getWriter(cfg.Path+string(filepath.Separator)+"error.log", maxBackupSize, maxAge, maxSize, cfg.GetCompress())

	// 输出多个
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(debugWriter), debugLevel),
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(infoWriter), infoLevel), //将info及以下写入logPath，NewConsoleEncoder 是非结构化输出
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(warnWriter), warnLevel), //warn及以上写入errPath
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(errorWriter), errorLevel),
		//zapcore.NewCore(zapcore.NewJSONEncoder(config), zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), logLevel), //同时将日志输出到控制台，NewJSONEncoder 是结构化输出
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	c = &Core{
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		logger: logger,
	}

	return c
}

func (c *Core) CoreLogger(ctx context.Context) log.Logger {
	return &entryCore{
		ctx:    ctx,
		pool:   c.pool,
		logger: c.logger,
	}
}

// *Helper
func (c *Core) WithContext(ctx context.Context) *log.Helper {
	traceId := getTraceId(ctx)
	logger := c.logger.With(zap.String("trace_id", traceId))

	return log.NewHelper(&entryCore{
		ctx:    ctx,
		pool:   c.pool,
		logger: logger,
	})
}

type entryCore struct {
	ctx    context.Context
	pool   *sync.Pool
	logger *zap.Logger
}

func (c *entryCore) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}

	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}

	buf := c.pool.Get().(*bytes.Buffer)

	for i := 0; i < len(keyvals); i += 2 {
		fmt.Fprintf(buf, " %s=%v", keyvals[i], keyvals[i+1])
	}

	switch level {
	case log.LevelDebug:
		c.logger.Debug(buf.String())
	case log.LevelInfo:
		c.logger.Info(buf.String())
	case log.LevelWarn:
		c.logger.Warn(buf.String())
	case log.LevelError:
		c.logger.Error(buf.String())
	default:
		c.logger.Debug(buf.String())

	}

	buf.Reset()
	c.pool.Put(buf)
	return nil
}

func getWriter(filename string, maxBackup, maxAge, maxSize int, compress bool) io.Writer {
	fmt.Printf("getWriter %s, maxBackup:%d, maxAge:%d, maxSize:%dmb, compress:%v\n", filename, maxBackup, maxAge, maxSize, compress)
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,   //the file max size, unit is mb, if overflow the file will rotate
		MaxBackups: maxBackup, // 最大文件保留数, 超过就删除最老的日志文件
		MaxAge:     maxAge,    // 保留文件的最大天数
		Compress:   compress,  // 不启用压缩的功能
		LocalTime:  true,      // 本地时间分割
	}
}

// get trace id
func getTraceId(ctx context.Context) string {
	span := opentracing.SpanFromContext(ctx)
	var traceID string
	if span == nil {
		traceID = ""
	} else {
		traceID = strings.SplitN(fmt.Sprintf("%s", span.Context()), ":", 2)[0]
	}
	return traceID
}
