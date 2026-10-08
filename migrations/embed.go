// Package migrations embeds the SQL migrations of every module, applied in file name order.
package migrations

import "embed"

// The runner ignores non-.sql entries such as this file.
//
//go:embed *
var FS embed.FS
