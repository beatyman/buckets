package cmd

import (
	"fmt"
	"os"
	"os/signal"
)

func Message(format string, args ...interface{}) {

}

func Warn(format string, args ...interface{}) {

}

func Success(format string, args ...interface{}) {

}

func JSON(data interface{}) {

}

func End(format string, args ...interface{}) {
	Message(format, args...)
	os.Exit(0)
}

func Error(err error, args ...interface{}) {

}

func Fatal(err error, args ...interface{}) {
	Error(err, args...)
	os.Exit(1)
}

func ErrCheck(err error, args ...interface{}) {
	if err != nil {
		Fatal(err, args...)
	}
}

func RenderTable(header []string, data [][]string) {
	fmt.Println()
	RenderTableWithoutNewLines(header, data)
	fmt.Println()
}

func RenderTableWithoutNewLines(header []string, data [][]string) {

}

func HandleInterrupt(stop func()) {
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
	stop()
	os.Exit(1)
}

func SetupDefaultLoggingConfig(file string) error {
	return nil
}
