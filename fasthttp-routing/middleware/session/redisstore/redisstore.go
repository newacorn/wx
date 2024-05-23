package redisstore

import (
	"context"
	"time"
	"unsafe"

	"github.com/redis/rueidis"
	"helpers/unsafefn"
)

type RedisStore struct {
	cli    rueidis.Client
	prefix string
}

func New(client rueidis.Client) *RedisStore {
	return NewWithPrefix(client, "scs:session:")
}

func NewWithPrefix(client rueidis.Client, prefix string) *RedisStore {

	return &RedisStore{cli: client, prefix: prefix}
}

//goland:noinspection ALL
func (r *RedisStore) Find(tokenWithPrefix []byte) (b []byte, exists bool, err error) {

	resp := r.cli.Do(context.Background(), r.cli.B().Get().Key(unsafefn.BtoS(tokenWithPrefix)).Build())
	if resp.Error() == rueidis.Nil {
		return nil, false, nil
	} else if resp.Error() != nil {
		return nil, false, resp.Error()
	}
	b, err = resp.AsBytes()
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

//goland:noinspection GoDirectComparisonOfErrors
func (r *RedisStore) FindCtx(ctx context.Context, tokenWithPrefix []byte) (b []byte, exists bool, err error) {

	resp := r.cli.Do(ctx, r.cli.B().Get().Key(r.prefix+string(tokenWithPrefix)).Build())
	if resp.Error() == rueidis.Nil {
		return nil, false, nil
	} else if resp.Error() != nil {
		return nil, false, resp.Error()
	}
	b, err = resp.AsBytes()
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

func (r *RedisStore) Commit(tokenWithPrefix []byte, encodedData []byte, expiry time.Time, modified bool) (err error) {
	if !modified {
		var count int64
		result := r.cli.Do(context.Background(), r.cli.B().Expireat().Key(unsafefn.BtoS(tokenWithPrefix)).Timestamp(expiry.Unix()).Build())
		count, err = result.AsInt64()
		if err != nil || count != 0 {
			return err
		}
	}
	return r.cli.Do(context.Background(), r.cli.B().Set().Key(unsafefn.BtoS(tokenWithPrefix)).Value(unsafefn.BtoS(encodedData)).Exat(expiry).Build()).Error()
}
func (r *RedisStore) CommitCtx(ctx context.Context, tokenWithPrefix []byte, encodedData []byte, expiry time.Time,
	modified bool) (err error) {
	if !modified {
		var count int64
		result := r.cli.Do(context.Background(), r.cli.B().Expireat().Key(unsafefn.BtoS(tokenWithPrefix)).Timestamp(expiry.Unix()).Build())
		count, err = result.AsInt64()
		if err != nil || count != 0 {
			return err
		}
	}
	return r.cli.Do(ctx, r.cli.B().Set().Key(unsafefn.BtoS(tokenWithPrefix)).Value(unsafefn.BtoS(encodedData)).Exat(expiry).Build()).Error()
}
func (r *RedisStore) Delete(tokenWithPrefix []byte) error {
	if r.cli.Do(context.Background(), r.cli.B().Unlink().Key(r.prefix+string(tokenWithPrefix)).Build()).Error() != nil {
		return r.cli.Do(context.Background(), r.cli.B().Del().Key(r.prefix+string(tokenWithPrefix)).Build()).Error()
	}
	return nil
}
func (r *RedisStore) DeleteCtx(ctx context.Context, tokenWithPrefix []byte) error {
	return r.cli.Do(ctx, r.cli.B().Del().Key(unsafefn.BtoS(tokenWithPrefix)).Build()).Error()
}

func (r *RedisStore) All() (map[string][]byte, error) {
	all := make(map[string][]byte)
	ctx := context.Background()
	cursor := uint64(0)
	for {
		entry, err := r.cli.Do(ctx, r.cli.B().Scan().Cursor(cursor).Match(r.prefix+"*").Count(1000).Build()).AsScanEntry()
		if err != nil {
			return nil, err
		}
		cursor = entry.Cursor
		if cursor == 0 {
			break
		}
		if len(entry.Elements) == 0 {
			continue
		}

		sega, err := r.cli.Do(ctx, r.cli.B().Mget().Key(entry.Elements...).Build()).AsStrSlice()
		if err != nil {
			return nil, err
		}
		for i := range sega {
			if sega[i] == "" {
				continue
			}
			all[entry.Elements[i][len(r.prefix):]] = unsafe.Slice(unsafe.StringData(sega[i]), len(sega[i]))
		}
	}
	if len(all) != 0 {
		return all, nil
	}
	return nil, nil
}
func (r *RedisStore) AllCtx(ctx context.Context) (map[string][]byte, error) {
	all := make(map[string][]byte)
	cursor := uint64(0)
	for {
		entry, err := r.cli.Do(ctx, r.cli.B().Scan().Cursor(cursor).Match(r.prefix+"*").Count(1000).Build()).AsScanEntry()
		if err != nil {
			return nil, err
		}

		if len(entry.Elements) == 0 {
			continue
		}

		sega, err := r.cli.Do(ctx, r.cli.B().Mget().Key(entry.Elements...).Build()).AsStrSlice()
		if err != nil {
			return nil, err
		}
		for i := range sega {
			if sega[i] == "" {
				continue
			}
			all[entry.Elements[i][len(r.prefix):]] = unsafe.Slice(unsafe.StringData(sega[i]), len(sega[i]))
		}
		cursor = entry.Cursor
		if cursor == 0 {
			break
		}
	}
	if len(all) != 0 {
		return all, nil
	}
	return nil, nil
}

// func makeMillisecondTimestamp(t time.Time) int64 {
// 	return t.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
// }
