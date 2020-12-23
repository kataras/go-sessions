package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/kataras/go-sessions/v3"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var errPathMissing = errors.New("path is required")

// Database the BoltDB(file-based) session storage.
type Database struct {
	mongo *mongo.Collection
}

// New creates and returns a new MongoDB(file-based) storage with custom client options.
// Database and collection names should be included.
//
// It will remove any old session files.
func New(clientOpts *options.ClientOptions, databaseName, collectionName string) (*Database, error) {
	// var cred options.Credential
	// cred.AuthSource = "YourAuthSource"
	// cred.Username = "YourUserName"
	// cred.Password = "YourPassword"

	// clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_DB_URI")).SetAuth(cred)

	ctx := context.Background()
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	collection := client.Database(databaseName).Collection(collectionName)

	return NewFromDB(collection), nil
}

// NewFromDB same as `New` but accepts an already-created custom mongo collection client connection instead.
func NewFromDB(collection *mongo.Collection) *Database {
	db := &Database{mongo: collection}
	return db
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

}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	return nil
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {

}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	return 0
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	return false
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {

}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {

}

// Close terminates Dgraph's gRPC connection.
func (db *Database) Close() error {
	return nil
}
