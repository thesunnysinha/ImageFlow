// Package migrations embeds the versioned SQL files applied at start, in file-name order.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
