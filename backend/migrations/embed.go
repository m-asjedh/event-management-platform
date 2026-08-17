// Package migrations carries the SQL files into the binary.
//
// The embed lives beside the .sql files rather than in cmd/migrate, because go:embed
// cannot reach into a parent directory. It also means the files sit at the root of the
// FS, which is where goose looks.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
