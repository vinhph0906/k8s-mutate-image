package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/sqooba/k8s-mutate-image-and-policy/configs"
	"gopkg.in/natefinch/lumberjack.v2"
)

// SetupLogger configures and returns a logrus logger based on the configuration
func SetupLogger(cfg *configs.Log) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
		logger.Warnf("Invalid log level '%s', defaulting to 'info'", cfg.Level)
	} else {
		logger.SetLevel(level)
	}

	// Set log format
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
		logger.Warnf("Invalid log format '%s', defaulting to 'text'", cfg.Format)
	}

	// Set output destination
	switch cfg.Output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "file":
		if cfg.FilePath != "" {
			fileWriter := setupFileWriter(cfg)
			logger.SetOutput(fileWriter)
		} else {
			logger.SetOutput(os.Stdout)
			logger.Warn("Log file path not specified, falling back to stdout")
		}
	case "both":
		if cfg.FilePath != "" {
			fileWriter := setupFileWriter(cfg)
			multiWriter := io.MultiWriter(os.Stdout, fileWriter)
			logger.SetOutput(multiWriter)
		} else {
			logger.SetOutput(os.Stdout)
			logger.Warn("Log file path not specified, falling back to stdout only")
		}
	default:
		logger.SetOutput(os.Stdout)
		logger.Warnf("Invalid log output '%s', defaulting to stdout", cfg.Output)
	}

	return logger
}

// setupFileWriter creates a file writer with log rotation using lumberjack
func setupFileWriter(cfg *configs.Log) io.Writer {
	// Ensure the log directory exists
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logrus.Errorf("Failed to create log directory '%s': %v", logDir, err)
		return os.Stdout
	}

	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,    // megabytes
		MaxBackups: cfg.MaxBackups, // number of backups
		MaxAge:     cfg.MaxAge,     // days
		Compress:   cfg.Compress,   // compress old files
		LocalTime:  true,           // use local time for backup file names
	}
}
