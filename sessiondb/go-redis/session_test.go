package redis

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/kataras/go-sessions/v3"
	"github.com/stretchr/testify/assert"
)

func TestSessions_Acquire(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid     string
		expires time.Duration
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
		after  func(*testing.T, *Sessions, args)
	}{
		{
			name: "key does not exist",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectTTL("test:sid").SetVal(-2)
				mock.ExpectHSet("test:sid", "test:sid", true).SetVal(1)
				mock.ExpectExpire("test:sid", time.Minute).SetVal(true)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:     "sid",
				expires: time.Minute,
			},
			after: func(t *testing.T, s *Sessions, a args) {
				assert.Equal(t, sessions.LifeTime{}, s.Acquire(a.sid, a.expires))
			},
		},
		{
			name: "value does not exist",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectTTL("test:sid").SetVal(-1)
				mock.ExpectHSet("test:sid", "test:sid", true).SetVal(1)
				mock.ExpectExpire("test:sid", time.Minute).SetVal(true)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:     "sid",
				expires: time.Minute,
			},
			after: func(t *testing.T, s *Sessions, a args) {
				assert.Equal(t, sessions.LifeTime{}, s.Acquire(a.sid, a.expires))
			},
		},
		{
			name: "nil return value",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectTTL("test:sid").RedisNil()
				mock.ExpectHSet("test:sid", "test:sid", true).SetVal(1)
				mock.ExpectExpire("test:sid", time.Minute).SetVal(true)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:     "sid",
				expires: time.Minute,
			},
			after: func(t *testing.T, s *Sessions, a args) {
				assert.Equal(t, sessions.LifeTime{}, s.Acquire(a.sid, a.expires))
			},
		},
		{
			name: "with error",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectTTL("test:sid").SetErr(nil)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:     "sid",
				expires: time.Minute,
			},
			after: func(t *testing.T, s *Sessions, a args) {
				assert.Equal(t, sessions.LifeTime{Time: sessions.CookieExpireDelete}, s.Acquire(a.sid, a.expires))
			},
		},
		{
			name: "existing key",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectTTL("test:sid").SetVal(time.Minute)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:     "sid",
				expires: time.Minute,
			},
			after: func(t *testing.T, s *Sessions, a args) {
				l := s.Acquire(a.sid, a.expires)
				assert.IsType(t, sessions.LifeTime{}, l)
				assert.Greater(t, l.Time, time.Now().UTC())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			tt.after(t, s, tt.args)
		})
	}
}

func TestSessions_OnUpdateExpiration(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid        string
		expiration time.Duration
	}
	tests := []struct {
		name      string
		fields    func(*testing.T) fields
		args      args
		assertion assert.ErrorAssertionFunc
	}{
		{
			name: "unable to update expiry",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectExpire("test:sid", time.Minute).SetErr(errors.New("nope"))
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:        "sid",
				expiration: time.Minute,
			},
			assertion: func(t assert.TestingT, err error, vals ...interface{}) bool {
				assert.ErrorContains(t, err, "nope")
				return true
			},
		},
		{
			name: "successfully update expiry",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectExpire("test:sid", time.Minute).SetVal(true)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:        "sid",
				expiration: time.Minute,
			},
			assertion: func(t assert.TestingT, err error, vals ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			tt.assertion(t, s.OnUpdateExpiration(tt.args.sid, tt.args.expiration))
		})
	}
}

func TestSessions_Set(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid       string
		lifetime  sessions.LifeTime
		field     string
		value     interface{}
		immutable bool
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
	}{
		{
			name: "immutuable set success",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHSetNX("test:sid", "cool", true).SetVal(true)
				// mock.ExpectExpire("test:sid", time.Minute).SetVal(true) // NOTE: Unable to match this value for now
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:       "sid",
				lifetime:  sessions.LifeTime{Time: time.Now().Add(time.Minute)},
				field:     "cool",
				value:     true,
				immutable: true,
			},
		},
		{
			name: "mutuable set success",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHSet("test:sid", "cool", true).SetVal(1)
				// mock.ExpectExpire("test:sid", time.Minute).SetVal(true) // NOTE: Unable to match this value for now
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:       "sid",
				lifetime:  sessions.LifeTime{Time: time.Now().Add(time.Minute)},
				field:     "cool",
				value:     true,
				immutable: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			s.Set(tt.args.sid, tt.args.lifetime, tt.args.field, tt.args.value, tt.args.immutable)
		})
	}
}

func TestSessions_Get(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid   string
		field string
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
		want   interface{}
	}{
		{
			name: "nil value",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGet("test:sid", "cool").RedisNil()
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:   "sid",
				field: "cool",
			},
			want: nil,
		},
		{
			name: "error",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGet("test:sid", "cool").SetErr(errors.New("uncool"))
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:   "sid",
				field: "cool",
			},
			want: nil,
		},
		{
			name: "actual vlue",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGet("test:sid", "cool").SetVal("nice")
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:   "sid",
				field: "cool",
			},
			want: "nice",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			assert.Equal(t, tt.want, s.Get(tt.args.sid, tt.args.field))
		})
	}
}

func TestSessions_Visit(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid string
		cb  func(field string, value interface{})
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   func(*testing.T) args
	}{
		{
			name: "missing key",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGetAll("test:sid").RedisNil()
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: func(t *testing.T) args {
				return args{
					sid: "sid",
					cb: func(field string, value interface{}) {
						assert.Fail(t, "this should not have been called")
					},
				}
			},
		},
		{
			name: "error",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGetAll("test:sid").SetErr(errors.New("whoops"))
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: func(t *testing.T) args {
				return args{
					sid: "sid",
					cb: func(field string, value interface{}) {
						assert.Fail(t, "this should not have been called")
					},
				}
			},
		},
		{
			name: "calls callback with values",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHGetAll("test:sid").SetVal(map[string]string{"data": "secret"})
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: func(t *testing.T) args {
				ch := make(chan string, 1)

				go func() {
					select {
					case val := <-ch:
						assert.Equal(t, "data:secret", val)
					case <-time.After(time.Second):
						assert.Fail(t, "callback timeout")
					}
				}()

				return args{
					sid: "sid",
					cb: func(field string, value interface{}) {
						ch <- fmt.Sprintf("%s:%v", field, value)
					},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			args := tt.args(t)
			s.Visit(args.sid, args.cb)
		})
	}
}

func TestSessions_Len(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid string
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
		want   int
	}{
		{
			name: "error",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHLen("test:sid").SetErr(errors.New("oh no"))
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid: "sid",
			},
			want: 0,
		},
		{
			name: "it subtracts one",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHLen("test:sid").SetVal(4)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid: "sid",
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			assert.Equal(t, tt.want, s.Len(tt.args.sid))
		})
	}
}

func TestSessions_Delete(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid   string
		field string
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
		want   bool
	}{
		{
			name: "error",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHDel("test:sid", "gun").SetErr(errors.New("hmm"))
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:   "sid",
				field: "gun",
			},
			want: false,
		},
		{
			name: "success",
			fields: func(*testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHDel("test:sid", "bun").SetVal(1)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid:   "sid",
				field: "bun",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			assert.Equal(t, tt.want, s.Delete(tt.args.sid, tt.args.field))
		})
	}
}

func TestSessions_Clear(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid string
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
	}{
		{
			name: "removes our special tracking key",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectHKeys("test:sid").SetVal([]string{"size", "colour", "test:sid"})
				mock.ExpectHDel("test:sid", "size", "colour").SetVal(2)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid: "sid",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			s.Clear(tt.args.sid)
		})
	}
}

func TestSessions_Release(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	type args struct {
		sid string
	}
	tests := []struct {
		name   string
		fields func(*testing.T) fields
		args   args
	}{
		{
			name: "deletes the hash key",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				mock.ExpectDel("test:sid").SetVal(1)
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			args: args{
				sid: "sid",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			s.Release(tt.args.sid)
		})
	}
}

func TestSessions_Close(t *testing.T) {
	config := &Config{
		Prefix: "test",
	}

	type fields struct {
		client *redis.Client
		config *Config
	}
	tests := []struct {
		name      string
		fields    func(*testing.T) fields
		assertion assert.ErrorAssertionFunc
	}{
		{
			name: "closes the client",
			fields: func(t *testing.T) fields {
				db, mock := redismock.NewClientMock()
				t.Cleanup(func() {
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Error(err)
					}
				})

				return fields{
					client: db,
					config: config,
				}
			},
			assertion: func(t assert.TestingT, err error, val ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := tt.fields(t)
			s := &Sessions{
				client: fields.client,
				config: fields.config,
			}
			tt.assertion(t, s.Close())
		})
	}
}
