package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// WatchdogConfig configuration for a watchdog.
type WatchdogConfig struct {
	TCPPort          int
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	ExecTimeout      time.Duration

	FunctionProcess    string
	ContentType        string
	InjectCGIHeaders   bool
	OperationalMode    int
	LogBufferSizeBytes int
	LogLevel           log.Level
}

// Process returns a string for the process and a slice for the arguments from the FunctionProcess.
func (w WatchdogConfig) Process() (string, []string) {
	parts := strings.Split(w.FunctionProcess, " ")

	if len(parts) > 1 {
		return parts[0], parts[1:]
	}

	return parts[0], []string{}
}

// New create config based upon environmental variables.
func New(env []string) (WatchdogConfig, error) {
	log.SetFormatter(&log.JSONFormatter{})
	envMap := mapEnv(env)

	var functionProcess string
	if val, exists := envMap["fprocess"]; exists {
		functionProcess = val
	}

	if val, exists := envMap["function_process"]; exists {
		functionProcess = val
	}

	contentType := "application/octet-stream"
	if val, exists := envMap["content_type"]; exists {
		contentType = val
	}

	config := WatchdogConfig{
		TCPPort:            getInt(envMap, "port", 8080),
		HTTPReadTimeout:    getDuration(envMap, "read_timeout", time.Second*10),
		HTTPWriteTimeout:   getDuration(envMap, "write_timeout", time.Second*10),
		FunctionProcess:    functionProcess,
		InjectCGIHeaders:   true,
		ExecTimeout:        getDuration(envMap, "exec_timeout", time.Second*10),
		OperationalMode:    ModeStreaming,
		ContentType:        contentType,
		LogBufferSizeBytes: getInt(envMap, "buffer_size", 1024),
		LogLevel:           getLogLevel(envMap, "log_level"),
	}

	if val := envMap["mode"]; len(val) > 0 {
		config.OperationalMode = WatchdogModeConst(val)
	}

	return config, nil
}

func mapEnv(env []string) map[string]string {
	mapped := map[string]string{}

	for _, val := range env {
		parts := strings.Split(val, "=")
		if len(parts) < 2 {
			fmt.Println("Bad environment: " + val)
		}
		mapped[parts[0]] = parts[1]
	}

	return mapped
}

func getDuration(env map[string]string, key string, defaultValue time.Duration) time.Duration {
	result := defaultValue
	if val, exists := env[key]; exists {
		parsed, _ := time.ParseDuration(val)
		result = parsed

	}
	return result
}

func getInt(env map[string]string, key string, defaultValue int) int {
	result := defaultValue
	if val, exists := env[key]; exists {
		parsed, _ := strconv.Atoi(val)
		result = parsed
	}

	return result
}

func getLogLevel(env map[string]string, key string) log.Level {
	level := "info"
	if val, exists := env[key]; exists {
		level = strings.ToLower(val)
	}
	switch level {
	case "info":
		return log.InfoLevel
	case "debug":
		return log.DebugLevel
	case "error":
		return log.ErrorLevel
	default:
		log.Errorf("Unknown log_level - defaulting to INFO")
		return log.InfoLevel
	}

}
