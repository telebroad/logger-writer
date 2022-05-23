package logger

import (
	"log"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {

	myLogger, err := New("logs", "error-2006-01-02.log", 24, 30, 2)
	if err != nil {
		t.Fatalf("error writing to file %s", err.Error())
	}
	log.SetOutput(myLogger)

	log.Println("test-1")
	log.Println("test-2")
	log.Println("test-3")
	<-time.After(time.Second * 3)
	log.Println("test-4")
	log.Println("test-5")
	log.Println("test-6")
}
