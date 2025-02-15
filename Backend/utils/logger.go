package utils

import (
	"github.com/sirupsen/logrus"
	"os"
)

var (
	InfoLogger  *logrus.Logger
	ErrorLogger *logrus.Logger
)

func InitLogger() {
	InfoLogger = logrus.New()
	ErrorLogger = logrus.New()

	// Set output untuk InfoLogger ke stdout
	InfoLogger.SetOutput(os.Stdout)
	InfoLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set output untuk ErrorLogger ke stderr
	ErrorLogger.SetOutput(os.Stderr)
	ErrorLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Anda juga bisa menambahkan level logging
	InfoLogger.SetLevel(logrus.InfoLevel)
	ErrorLogger.SetLevel(logrus.ErrorLevel)
}
