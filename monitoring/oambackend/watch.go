package oambackend

import (
	"context"
	"io"
	"time"
)

func Watch(ctx context.Context, interval time.Duration, w io.Writer, renderFunc func() error) error {
	// Server-side watch is disabled to prevent blocking the HTTP handler.
	// The watch loop is now handled by the client.
	return renderFunc()
}
