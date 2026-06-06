package tui

import "errors"

// ErrNotTTY is returned when interactive search requires a terminal.
var ErrNotTTY = errors.New("interactive search requires a terminal (try hx search without -i)")
