package logger

import (
	"fmt"
	"testing"
)

func TestLogger(t *testing.T) {

	logr := New("logs", "error-2006-01-02.log", 24, 30)
	fmt.Fprintln(logr, "test-1")
	fmt.Fprintln(logr, "test-2")
	fmt.Fprintln(logr, "test-3")
}
