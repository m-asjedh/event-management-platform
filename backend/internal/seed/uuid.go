package seed

import "github.com/google/uuid"

// uuidv7 returns a version-7 UUID whose timestamp is fixed so a seed rerun
// produces the same keys. n is a monotonic counter, not a clock.
//
// IDs are generated here instead of relying on DEFAULT uuidv7(): COPY needs
// parent keys in Go before child rows (rooms/sessions point at events), and
// a database-assigned default would make reruns non-reproducible.
func uuidv7(n uint64) uuid.UUID {
	const epochMS = 1767225600000 // 2026-01-01T00:00:00Z
	var u uuid.UUID
	ts := epochMS + n
	u[0] = byte(ts >> 40)
	u[1] = byte(ts >> 32)
	u[2] = byte(ts >> 24)
	u[3] = byte(ts >> 16)
	u[4] = byte(ts >> 8)
	u[5] = byte(ts)
	u[6] = 0x70
	u[7] = 0
	u[8] = 0x80
	u[9] = byte(n >> 32)
	u[10] = byte(n >> 24)
	u[11] = byte(n >> 16)
	u[12] = byte(n >> 8)
	u[13] = byte(n)
	u[14] = 0
	u[15] = 0
	return u
}
