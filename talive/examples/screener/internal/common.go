package internal

import (
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	_ = godotenv.Load(path.Join(pwd, ".env"))
}

func LogLevel() string {
	return strings.ToLower(env("LOG_LEVEL", "INFO"))
}

func LogFormat() string {
	return strings.ToLower(env("LOG_FORMAT", "text"))
}

func env(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewLogger() *slog.Logger {
	options := &slog.HandlerOptions{
		Level: parseLogLevel(LogLevel()),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().UTC().Format("2006-01-02T15:04:05.000000Z"))
			}
			return a
		},
	}

	var handler slog.Handler
	if LogFormat() == "json" {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
