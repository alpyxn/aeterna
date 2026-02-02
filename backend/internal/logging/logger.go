package logging

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Init() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	format := strings.ToLower(os.Getenv("LOG_FORMAT"))

	var output *lumberjack.Logger
	logFile := os.Getenv("LOG_FILE")
	if logFile != "" {
		output = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 50),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 14),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		}
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if format == "text" {
		if output != nil {
			handler = slog.NewTextHandler(output, handlerOpts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, handlerOpts)
		}
	} else {
		if output != nil {
			handler = slog.NewJSONHandler(output, handlerOpts)
		} else {
			handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
		}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}
