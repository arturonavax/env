package sqlite

import (
	"time"
)

// Options specifies configuration parameters when opening the SQLite storage engine.
type Options struct {
	// Path is the filesystem path to the database file (e.g. ~/.local/state/anx/anx.db).
	Path string

	// BusyTimeout controls how long SQLite waits when acquiring a busy database lock.
	// Default is 5 seconds.
	BusyTimeout time.Duration

	// Synchronous controls the fsync guarantee: "FULL" (default) or "NORMAL".
	Synchronous string

	// ReadOnly opens the database in read-only mode.
	ReadOnly bool

	// MaxReadConns configures the size of the reader connection pool. Default is 2.
	MaxReadConns int
}

func (o *Options) setDefaults() {
	if o.BusyTimeout <= 0 {
		o.BusyTimeout = 5 * time.Second
	}
	if o.Synchronous == "" {
		o.Synchronous = "FULL"
	}
	if o.MaxReadConns <= 0 {
		o.MaxReadConns = 2
	}
}
