package graylog_test

import (
	"time"

	"github.com/mdigger/graylog"
	"golang.org/x/exp/slog"
)

func Example() {
	// init graylog logger
	log, err := graylog.Dial("udp", "localhost:12201")
	if err != nil {
		panic(err)
	}
	defer log.Close()

	// send info message with attributes
	log.Info("Test message.\nMore info...",
		slog.Any("log", log),
		slog.Bool("bool", true),
		slog.Time("now", time.Now()),
		slog.Group("group",
			slog.String("str", "string value"),
			slog.Duration("duration", time.Hour/3)),
		slog.Any("struct", struct {
			Test string `json:"test"`
		}{Test: "test"}),
	)

	// register as default
	slog.SetDefault(log.Logger)
}
