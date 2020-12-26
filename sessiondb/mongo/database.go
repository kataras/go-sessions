package mongo

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/kataras/go-sessions/v3"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var errDatabaseNameMissing = errors.New("database name is required")

// Database the BoltDB(file-based) session storage.
type Database struct {
	mongo *mongo.Database
}

// New creates and returns a new MongoDB(file-based) storage with custom client options.
// Database and collection names should be included.
//
// It will remove any old session files.
func New(clientOpts *options.ClientOptions, database string) (*Database, error) {
	if database == "" {
		return nil, errDatabaseNameMissing
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	mongo := client.Database(database)
	return &Database{mongo: mongo}, nil
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	var result bson.Raw
	ctx := context.TODO()
	res := db.mongo.Collection(sid).FindOne(ctx, bson.D{{"key", sid}})

	// not found, create an entry and return an empty lifetime, session manager will do its job.
	if err := res.Err(); err != nil {
		expirationTime := time.Now().Add(expires)
		timeBytes, _ := sessions.DefaultTranscoder.Marshal(expirationTime)
		db.mongo.Collection(sid).InsertOne(
			context.TODO(),
			bson.D{{"$set", bson.D{{"key", sid}, {"value", timeBytes}}}},
		)

		return sessions.LifeTime{Time: sessions.CookieExpireDelete}
	}

	// found, return the expiration.
	res.Decode(&result)
	result.Validate()
	val := result.Lookup("value")
	var expirationTime time.Time
	sessions.DefaultTranscoder.Unmarshal(val.Value, &expirationTime)
	return sessions.LifeTime{Time: expirationTime}
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

	db.mongo.Collection(sid).UpdateOne(
		context.Background(),
		// filter
		bson.D{{"key", key}},
		// update
		bson.D{{"$set", bson.D{{"key", key}, {"value", valueBytes}}}},
		// options
		options.Update().SetUpsert(true),
	)

	// indexOpts := options.CreateIndexes().SetMaxTime(10 * time.Second)
	// dur := lifetime.DurationUntilExpiration()
	// index := mongo.IndexModel{
	// 	Keys:    bson.D{{Key: "expireAt", Value: 1}},
	// 	Options: options.Index().SetExpireAfterSeconds(int32(dur))}

	// db.mongo.Collection(sid).Indexes().CreateOne(context.Background(), index, indexOpts)
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	var result bson.Raw
	ctx := context.TODO()
	res := db.mongo.Collection(sid).FindOne(ctx, bson.D{{"key", key}})
	err := res.Decode(&result)
	if err != nil {
		return
	}

	err = result.Validate()
	if err != nil {
		return
	}

	val := result.Lookup("value")
	return sessions.DefaultTranscoder.Unmarshal(val.Value, &value)
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {
	ctx := context.TODO()
	res, err := db.mongo.Collection(sid).Find(ctx, bson.D{})
	if err != nil {
		return
	}

	for res.Next(context.TODO()) {
		var result bson.Raw
		if err := res.Decode(&result); err != nil {
			log.Fatal(err)
		}

		k := result.Lookup("key")
		v := result.Lookup("value")
		var val interface{}
		sessions.DefaultTranscoder.Unmarshal(v.Value, &val)
		cb(k.String(), val)
	}
	if err := res.Err(); err != nil {
		log.Fatal(err)
	}
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	ctx := context.TODO()
	number, err := db.mongo.Collection(sid).CountDocuments(ctx, bson.D{})
	if err == nil {
		n = int(number)
	}

	return
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	ctx := context.TODO()
	_, err := db.mongo.Collection(sid).DeleteOne(ctx, bson.D{{"key", key}})
	if err != nil {
		deleted = false
		return
	}
	deleted = true
	return
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	db.mongo.Collection(sid).DeleteMany(context.TODO(), bson.D{{"key", bson.D{{"$ne", sid}}}})
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {
	db.mongo.Collection(sid).Drop(context.TODO())
}

// Close terminates Dgraph's gRPC connection.
func (db *Database) Close() error {
	db.mongo.Client().Disconnect(context.TODO())
	return nil
}
