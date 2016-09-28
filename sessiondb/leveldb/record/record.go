package record // import "github.com/kataras/go-sessions/sessiondb/leveldb/record"

import "time"

// Record The structure written to the database
type Record struct {
	Data      []byte
	DeathTime time.Time
}
