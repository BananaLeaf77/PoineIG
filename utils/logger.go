package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger
var logFile *os.File

func InitLogger() {
	// Create logs directory
	os.MkdirAll("logs", 0755)

	// Log file with timestamp
	filename := fmt.Sprintf("logs/run_%s.log", time.Now().Format("2006-01-02_15-04-05"))
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	logFile = f

	// Pretty terminal logger
	Logger = log.New(os.Stdout)
	Logger.SetLevel(log.DebugLevel)
	Logger.SetTimeFormat("15:04:05")
	Logger.SetReportTimestamp(true)

	// Also write to file (plain text)
	fileLogger := log.New(f)
	fileLogger.SetLevel(log.DebugLevel)
	fileLogger.SetTimeFormat("2006-01-02 15:04:05")
	fileLogger.SetReportTimestamp(true)

	fmt.Printf("📝 Logging to %s\n\n", filename)
}

func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
