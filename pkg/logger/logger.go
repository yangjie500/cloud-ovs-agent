package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"

	"github.com/yangjie500/cloud-ovs-agent/pkg/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Level int32

const (
	Debug Level = iota
	Info
	Warn
	Error
)

var (
	currentLevel atomic.Int32
	logger       *log.Logger
)

func init() {
	config, err := config.LoadAll(".env")
	if err != nil {
		log.Fatalf("Error reading configuration: %v", err)
	}
	initWriters(config)
	SetLevelFromEnv(config)
}

func initWriters(cfg config.Config) {
	rot := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogMaxSizeMb,
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAgeDays,
		Compress:   cfg.LogCompress,
	}

	var w io.Writer = rot
	if cfg.LogToStdout {
		w = io.MultiWriter(os.Stdout, rot)
	}
	// Use UTC timestamps for consistency across hosts/regions.
	flags := log.LstdFlags | log.Lmicroseconds | log.Lshortfile | log.LUTC
	logger = log.New(w, "", flags)
}

func SetLevel(l Level) {
	currentLevel.Store(int32(l))
}

func SetLevelFromEnv(cfg config.Config) {
	levelStr := strings.ToLower(cfg.LogLevel)
	fmt.Println(levelStr)
	switch levelStr {
	case "debug":
		SetLevel(Debug)
	case "info":
		SetLevel(Info)
	case "warn":
		SetLevel(Warn)
	case "error":
		SetLevel(Error)
	default:
		SetLevel(Info)
	}
}

func logf(l Level, prefix, format string, args ...interface{}) {
	if l < Level(currentLevel.Load()) {
		return
	}

	msg := fmt.Sprintf(format, args...)
	logger.Output(4, fmt.Sprintf("[%s] %s", prefix, msg))
}

func Debugf(format string, args ...interface{}) {
	logf(Debug, "DEBUG", format, args...)
}

func Infof(format string, args ...interface{}) {
	logf(Info, "INFO", format, args...)
}

func Warnf(format string, args ...interface{}) {
	logf(Warn, "WARN", format, args...)
}

func Errorf(format string, args ...interface{}) {
	logf(Error, "ERROR", format, args...)
}
