package database

import (
	"proxy/config"
	"github.com/garyburd/redigo/redis"
	"time"
)

type DB struct {
	r *redis.Pool
}

func NewDB() *DB {
	conf := config.GetConfig()
	rdb := &redis.Pool{
		MaxIdle:     conf.RedisMaxIdle,
		MaxActive:   conf.RedisMaxActive,
		IdleTimeout: time.Duration(conf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", conf.RedisAddr,
				redis.DialConnectTimeout(time.Duration(3*time.Second)),
				redis.DialReadTimeout(time.Duration(3*time.Second)),
				redis.DialWriteTimeout(time.Duration(3*time.Second)))
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}
	db := &DB{
		r: rdb,
	}
	return db
}

func (db *DB) HGETString(key string, field interface{}) (s string, err error) {
	conn := db.r.Get()
	defer conn.Close()

	s, err = redis.String(conn.Do("HGET", key, field))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
	}
	return
}

func (db *DB) HGETUint64(key string, field interface{}) (n uint64, err error) {
	conn := db.r.Get()
	defer conn.Close()

	n, err = redis.Uint64(conn.Do("HGET", key, field))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
	}
	return
}

func (db *DB) SetPostInfo(key string,fieldUUID,uuid,fieldName,name,fieldParentId,pid interface{}) (err error) {
	conn := db.r.Get()
	defer conn.Close()

	_, err = conn.Do("HMSET", key,fieldUUID,uuid,fieldName,name,fieldParentId,pid)
	return
}

func (db *DB) SetAccount(key,fieldName,name,fieldPub,pubKey,fieldPri,priKey string) (err error) {
	conn := db.r.Get()
	defer conn.Close()

	_, err = conn.Do("HMSET", key,fieldName,name,fieldPub,pubKey,fieldPri,priKey)

	return
}

func (db *DB) HDEL(key string, arg interface{}) (deleteNum int, err error) {
	conn := db.r.Get()
	defer conn.Close()

	deleteNum, err = redis.Int(conn.Do("HDEL", key, arg))
	return
}

// func HEXISTS
func (db *DB) HEXISTS(key string, arg interface{}) (b bool, err error) {
	conn := db.r.Get()
	defer conn.Close()

	exist, err := redis.Int(conn.Do("HEXISTS", key, arg))
	if exist == 1 {
		b = true
	} else {
		b = false
	}
	return
}

func (db *DB) AddPostId(key string, postId uint64) (addNum int, err error) {
	conn := db.r.Get()
	defer conn.Close()

	addNum, err = redis.Int(conn.Do("SADD", key, postId))
	return
}

func (db *DB) SREM(key string, arg interface{}) (removeNum int, err error) {
	conn := db.r.Get()
	defer conn.Close()

	removeNum, err = redis.Int(conn.Do("SREM", key, arg))
	return
}

func (db *DB) SISMEMBER(key string, arg interface{}) (b bool, err error) {
	conn := db.r.Get()
	defer conn.Close()

	in, err := redis.Int(conn.Do("SISMEMBER", key, arg))
	if in == 1 {
		b = true
	} else {
		b = false
	}
	return
}

func (db *DB) EXISTS(key string) (b bool, err error) {
	conn := db.r.Get()
	defer conn.Close()

	exist, err := redis.Int(conn.Do("EXISTS", key))
	if exist == 1 {
		b = true
	} else {
		b = false
	}
	return
}

func (db *DB) SET(key string,arg interface{}) (err error) {
	conn := db.r.Get()
	defer conn.Close()

	_,err = conn.Do("SET", key, arg)
	return
}

func (db* DB) GETUint64(key string) (n uint64,err error) {
	conn := db.r.Get()
	defer conn.Close()

	n, err = redis.Uint64(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
	}
	return
}

func (db *DB) DEL(key string) (err error) {
	conn := db.r.Get()
	defer conn.Close()

	_,err = conn.Do("DEL", key)
	return
}

func (db *DB) MGETId(keys ...interface{}) (ids []string,err error) {
	conn := db.r.Get()
	defer conn.Close()

	ids,err = redis.Strings(conn.Do("MGET", keys...))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
	}
	if len(ids) == 1 && ids[0]== "" {
		ids = ids[:0]
	}
	return
}

func (db *DB) GETId(key interface{}) (id string,err error) {
	conn := db.r.Get()
	defer conn.Close()

	id,err = redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
	}
	return
}

func (db *DB) AddReward(key,fieldReward,reward interface{}) (err error) {
	conn := db.r.Get()
	defer conn.Close()

	_, err = conn.Do("HINCRBY", key,fieldReward,reward)
	return
}