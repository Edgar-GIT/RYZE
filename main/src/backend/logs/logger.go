package logs

import (
	"log/slog"
	"os"
)

// logger emits structured event records for auditable platform activity. It is
// intentionally tiny: it only wraps the standard library's slog handler so
// every record is a plain key/value line with a timestamp, level and message.
// A dedicated audit store does not exist yet; until one does, security-relevant
// events (currently the Test Mode lifecycle) are recorded here through the
// structured logging architecture.
var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// Info emits one structured event record. Callers must never pass passwords,
// raw tokens or any other secret as attributes.
func Info(event string, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr)
	}
	logger.Info(event, args...)
}