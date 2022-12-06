# graylog logger

Package graylog provides support for logging to the Graylog server.

It can send messages to the Graylog server using UDP or TCP.
When using UDP as a transport layer, the messages sent are gzip compressed
and automatically chunked.

```golang
import (
	"time"
	"github.com/mdigger/graylog"
	"golang.org/x/exp/slog"
)

func main() {
    // init graylog logger
    log, err := graylog.Dial("udp", "localhost:12201")
    if err != nil {
        panic(err)
    }
    defer log.Close()

    // send debug message with attributes
    log.Debug("Test message.\nMore info...",
        slog.Any("log", log),
        slog.Bool("bool", true),
        slog.Time("now", time.Now()),
        slog.Group("group",
            slog.String("str", "string value"),
            slog.Duration("duration", time.Hour/3)),
        slog.Any("object", struct {
            Text string `json:"text"`
        }{Text: "text"}),
    )
}
```
