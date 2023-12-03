package storage

import (
	"context"
	"errors"
	"time"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/redis/go-redis/v9"

	"github.com/ledisdb/ledisdb/ledis"
)

type Ledis struct {
	db *ledis.DB
}

func (l *Ledis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)

	res, err := l.db.Del(strsToBytes(keys)...)

	cmd.SetVal(res)
	cmd.SetErr(err)

	return cmd
}

func (l *Ledis) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)

	res, err := l.db.Get([]byte(key))

	cmd.SetVal(string(res))
	cmd.SetErr(err)

	return cmd
}
func (l *Ledis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)

	data, ok := value.(string)
	if !ok {
		cmd.SetErr(errors.New("只支持字符串"))
	}
	err := l.db.SetEX([]byte(key), int64(expiration), []byte(data))

	cmd.SetVal("ok")
	cmd.SetErr(err)

	return cmd
}
func (l *Ledis) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)

	if len(values)%2 != 0 {
		cmd.SetErr(errors.New("key 和 value 必须配对"))
		return cmd
	}

	pairs := []ledis.KVPair{}
	for i := 0; i < len(values); i += 2 {
		k := values[i]
		v := values[i+1]

		key, ok := k.(string)
		if !ok {
			cmd.SetErr(errors.New("只支持字符串"))
			return cmd
		}
		val, ok := v.(string)
		if !ok {
			cmd.SetErr(errors.New("只支持字符串"))
			return cmd
		}
		pairs = append(pairs, ledis.KVPair{
			Key:   []byte(key),
			Value: []byte(val),
		})
	}

	err := l.db.MSet(pairs...)

	cmd.SetVal("ok")
	cmd.SetErr(err)

	return cmd

}
func (l *Ledis) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	cmd := redis.NewSliceCmd(ctx)

	data, err := l.db.MGet(strsToBytes(keys)...)

	cmd.SetErr(err)
	cmd.SetVal(bytesToInterface(data))
	return cmd
}
func (l *Ledis) Close() error {
	return nil
}
func (l *Ledis) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("ok")
	cmd.SetErr(nil)
	return cmd
}

func NewLedis(cfg RedisConfig) (Redis, error) {
	config := lediscfg.NewConfigDefault()
	config.DataDir = cfg.DataDir
	l, _ := ledis.Open(config)
	db, _ := l.Select(0)

	return &Ledis{db: db}, nil
}

func strsToBytes(keys []string) (res [][]byte) {
	for _, k := range keys {
		res = append(res, []byte(k))
	}
	return
}

func bytesToInterface(keys [][]byte) (res []interface{}) {
	for _, k := range keys {
		res = append(res, k)
	}
	return
}

func MGet(ctx context.Context, r Redis, keys []string) (vals [][]byte, err error) {
	cmd := r.MGet(ctx, keys...)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	for _, v := range cmd.Val() {
		val := []byte(v.(string))
		vals = append(vals, val)
	}
	return
}

func MSet(ctx context.Context, r Redis, m map[string]interface{}) error {
	pairs := []interface{}{}
	for k, v := range m {
		val, ok := v.(string)
		if !ok {
			return errors.New("只支持字符串")
		}
		pairs = append(pairs, []byte(k), []byte(val))
	}

	cmd := r.MSet(ctx, pairs...)
	return cmd.Err()
}
