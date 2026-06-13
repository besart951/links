package migrations

import "embed"

// FS contains SQL migrations for goose.
//
//go:embed *.sql
var FS embed.FS
