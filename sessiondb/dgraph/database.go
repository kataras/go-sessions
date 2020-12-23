package dgraph

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/kataras/go-sessions/v3"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
)

var errPathMissing = errors.New("path is required")

// Database the BoltDB(file-based) session storage.
type Database struct {
	dg   *dgo.Dgraph
	conn *grpc.ClientConn
}

// New creates and returns a new Dgraph database connection to "target" with `grpc.WithInsecure()`.
// Target should include the url to Dgraph's alpha gRPC-external-public port.
//
// It will remove any old session files.
func New(target string) (*Database, error) {
	if target == "" {
		return nil, errPathMissing
	}

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return NewFromDB(conn)
}

// NewFromDB same as `New` but accepts an already-created secured gRPC connection instead.
func NewFromDB(conn *grpc.ClientConn) (*Database, error) {
	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	// check if schema already exists
	ctx := context.Background()
	query := `schema(type: SessionEntry) {}`
	response, err := dg.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var r struct {
		Schema []struct {
			Name   string `json:"name"`
			Fields []struct {
				Name string `json:"name"`
			} `json:"fields"`
		} `json:"types"`
	}

	err = json.Unmarshal(response.Json, &r)
	if err != nil {
		return nil, err
	}

	// if not then set schema
	if len(r.Schema) == 0 {
		op := &api.Operation{}
		op.Schema = `
	sid: string @index(hash) . 
	key: string @index(hash) . 
	value: string . 
	type SessionEntry {
		sid
		key
		value
	}
	`
		err := dg.Alter(ctx, op)
		if err != nil {
			return nil, err
		}
	}

	db := &Database{dg: dg, conn: conn}
	return db, nil
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	// found, return the expiration.

	// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.

	// create it and set the expiration, we don't care about the value there.

	return sessions.LifeTime{} // session manager will handle the rest.
}

// OnUpdateExpiration not implemented here, yet.
// Note that this error will not be logged, callers should catch it manually.
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return sessions.ErrNotImplemented
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, lifetime sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		return
	}

	ctx := context.Background()

	query := `
	{
		  q(func: eq(key, "` + key + `")) @filter(eq(sid, "` + sid + `"))  {
			v as uid
		  }
	}
`

	mutation := `
	uid(v) <sid> "` + sid + `" .
	uid(v) <key> "` + key + `" .
	uid(v) <value> "` + string(valueBytes) + `" . 
	uid(v) <dgraph.type> "SessionEntry" . 
	`

	req := &api.Request{
		Query: query,
		Mutations: []*api.Mutation{
			{
				SetNquads: []byte(mutation),
			},
		},
		CommitNow: true,
	}

	_, err = db.dg.NewTxn().Do(ctx, req)

	return

}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	ctx := context.Background()

	query := `{
	q(func: eq(key, "` + key + `")) @filter(eq(sid, "` + sid + `")) {
	  value
	}
}`

	response, err := db.dg.NewTxn().Query(ctx, query)
	if err != nil {
		return err
	}

	var r struct {
		Session []struct {
			// Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"q"`
	}

	err = json.Unmarshal(response.Json, &r)
	if err != nil {
		return err
	}

	if len(r.Session) == 0 {
		// not found.
		return nil
	}

	return sessions.DefaultTranscoder.Unmarshal([]byte(r.Session[0].Value), &value)
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {
	ctx := context.Background()

	query := `{
	q(func: eq(sid, "` + sid + `")) {
	  key
	  value
	}
}`

	response, err := db.dg.NewTxn().Query(ctx, query)
	if err != nil {
		return
	}

	var r struct {
		Session []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"q"`
	}

	err = json.Unmarshal(response.Json, &r)
	if err != nil {
		return
	}

	if len(r.Session) == 0 {
		// nothing found.
		return
	}

	for _, session := range r.Session {
		var value interface{}
		if err := sessions.DefaultTranscoder.Unmarshal([]byte(session.Value), &value); err != nil {
			return
		}

		cb(session.Key, value)
	}
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	ctx := context.Background()

	query := `{
	q(func: eq(sid, "` + sid + `")) {
	  count(uid)
	}
}`

	response, err := db.dg.NewTxn().Query(ctx, query)
	if err != nil {
		return
	}

	var r struct {
		Session []struct {
			TotalKeys int `json:"count"`
		} `json:"q"`
	}

	err = json.Unmarshal(response.Json, &r)
	if err == nil {
		n = r.Session[0].TotalKeys
		return
	}

	return
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	ctx := context.Background()

	query := `{
		  q(func: eq(sid, "` + sid + `")) {
			v as uid
		  }
}`

	deletion := `uid(v) "` + key + `" * . `

	req := &api.Request{
		Query: query,
		Mutations: []*api.Mutation{
			{
				DelNquads: []byte(deletion),
			},
		},
		CommitNow: true,
	}

	_, err := db.dg.NewTxn().Do(ctx, req)
	if err != nil {
		return false
	}

	return true
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	ctx := context.Background()

	query := `{
	q(func: eq(sid, "` + sid + `")) {
	  uid
	  key
	  value
	}
}`

	response, err := db.dg.NewTxn().Query(ctx, query)
	if err != nil {
		return
	}

	var r struct {
		Entries []struct {
			UID   string `json:"uid"`
			Key   string `json:"key"`
			Value string `json:"value"`
			Sid   string `json:"sid"`
		} `json:"q"`
	}

	err = json.Unmarshal(response.Json, &r)
	if err != nil {
		return
	}

	for _, entry := range r.Entries {
		// do not delete sid key entry
		if entry.Key == sid {
			continue
		}

		entry.Sid = sid
		del, _ := json.Marshal(entry)
		mu := &api.Mutation{
			CommitNow:  true,
			DeleteJson: del,
		}
		db.dg.NewTxn().Mutate(ctx, mu)
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {
	ctx := context.Background()

	query := `{
		  q(func: eq(sid, "` + sid + `")) {
			v as uid
		  }
}`

	deletion := `uid(v) * * . `

	req := &api.Request{
		Query: query,
		Mutations: []*api.Mutation{
			{
				DelNquads: []byte(deletion),
			},
		},
		CommitNow: true,
	}

	_, err := db.dg.NewTxn().Do(ctx, req)
	if err != nil {
		return
	}
}

// Close terminates Dgraph's gRPC connection.
func (db *Database) Close() error {
	return db.conn.Close()
}
