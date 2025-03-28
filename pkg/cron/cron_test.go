package cron

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/charmbracelet/log/v2"
)

func TestCronLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf)
	logger.SetLevel(log.DebugLevel)
	clogger := cronLogger{logger}
	clogger.Info("foo")
	clogger.Error(fmt.Errorf("bar"), "test")
	if buf.String() != "DEBU foo\nERRO test err=bar\n" {
		t.Errorf("unexpected log output: %s", buf.String())
	}
}

func TestSchedularAddRemove(t *testing.T) {
	s := NewScheduler(context.TODO())
	id, err := s.AddFunc("* * * * *", func() {})
	if err != nil {
		t.Fatal(err)
	}
	s.Remove(id)
}
