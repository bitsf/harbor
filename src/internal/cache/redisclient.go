package cache

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"
)

var knownPool sync.Map
var m sync.Mutex

// RedisPoolParam ...
type RedisPoolParam struct {
	PoolMaxIdle     int
	PoolMaxActive   int
	PoolIdleTimeout time.Duration

	DialConnectionTimeout time.Duration
	DialReadTimeout       time.Duration
	DialWriteTimeout      time.Duration
}

// GetRedisPool get a named redis pool
// supported rawurl
// redis://user:pass@redis_host:port/db
// redis+sentinel://user:pass@redis_sentinel1:port1,redis_sentinel2:port2/monitor_name/db?sentinel_password=pass
func GetRedisPool(name string, rawurl string, param *RedisPoolParam) (*redis.Pool, error) {
	if p, ok := knownPool.Load(name); ok {
		return p.(*redis.Pool), nil
	}
	m.Lock()
	defer m.Unlock()
	// load again in case multi threads
	if p, ok := knownPool.Load(name); ok {
		return p.(*redis.Pool), nil
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("bad redis url: %s", err)
	}

	if param == nil {
		param = &RedisPoolParam{
			PoolMaxIdle:           0,
			PoolMaxActive:         1,
			PoolIdleTimeout:       time.Minute,
			DialConnectionTimeout: time.Second,
			DialReadTimeout:       time.Second,
			DialWriteTimeout:      time.Second,
		}
	}
	if t := u.Query().Get("idle_timeout_seconds"); t != "" {
		if tt, e := strconv.Atoi(t); e == nil {
			param.PoolIdleTimeout = time.Second * time.Duration(tt)
		}
	}

	fmt.Println("get redis pool:", rawurl)
	if u.Scheme == "redis" {
		pool := &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(rawurl)
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
			MaxIdle:     param.PoolMaxIdle,
			MaxActive:   param.PoolMaxActive,
			IdleTimeout: param.PoolIdleTimeout,
			Wait:        true,
		}
		return pool, nil
	} else if u.Scheme == "redis+sentinel" {
		return getSentinelPool(u, param, err, name)
	} else if strings.HasPrefix(u.Scheme, "rediss") {
		return nil, fmt.Errorf("bad redis url: not support secure redis")
	} else {
		return nil, fmt.Errorf("bad redis url: not support scheme %s", u.Scheme)
	}
}

// redis+sentinel://user:pass@redis_sentinel1:port1,redis_sentinel2:port2/monitor_name/db?sentinel_password=pass
func getSentinelPool(u *url.URL, param *RedisPoolParam, err error, name string) (*redis.Pool, error) {
	ps := strings.Split(u.Path, "/")
	if len(ps) < 2 {
		return nil, fmt.Errorf("bad redis sentinel url: no master name")
	}

	var sentinelOptions []redis.DialOption
	if param.DialConnectionTimeout > 0 {
		sentinelOptions = append(sentinelOptions, redis.DialConnectTimeout(param.DialConnectionTimeout))
	}
	if param.DialReadTimeout > 0 {
		sentinelOptions = append(sentinelOptions, redis.DialReadTimeout(param.DialReadTimeout))
	}
	if param.DialWriteTimeout > 0 {
		sentinelOptions = append(sentinelOptions, redis.DialWriteTimeout(param.DialWriteTimeout))
	}

	redisOptions := sentinelOptions

	sentinelPassword := u.Query().Get("sentinel_password")
	if sentinelPassword != "" {
		sentinelOptions = append(sentinelOptions, redis.DialPassword(sentinelPassword))
	} else {
		if u.User != nil {
			password, isSet := u.User.Password()
			if isSet {
				sentinelOptions = append(sentinelOptions, redis.DialPassword(password))
			}
		}
	}
	if u.User != nil {
		password, isSet := u.User.Password()
		if isSet {
			redisOptions = append(redisOptions, redis.DialPassword(password))
		}
	}

	// sentinel doesn't need select db
	db := 0
	if len(ps) > 2 {
		db, err = strconv.Atoi(ps[2])
		if err != nil {
			return nil, fmt.Errorf("invalid database: %s", ps[1])
		}
		if db != 0 {
			redisOptions = append(redisOptions, redis.DialDatabase(db))
		}
	}

	sntnl := &sentinel.Sentinel{
		Addrs:      strings.Split(u.Host, ","),
		MasterName: ps[1],
		Dial: func(addr string) (redis.Conn, error) {
			fmt.Println("dial redis sentinel:", addr)
			c, err := redis.Dial("tcp", addr, sentinelOptions...)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}

	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			masterAddr, err := sntnl.MasterAddr()
			if err != nil {
				return nil, err
			}
			fmt.Println("dial redis master:", masterAddr, "db:", db)
			return redis.Dial("tcp", masterAddr, redisOptions...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if !sentinel.TestRole(c, "master") {
				return errors.New("role check failed")
			}
			return nil
		},
		MaxIdle:     param.PoolMaxIdle,
		MaxActive:   param.PoolMaxActive,
		IdleTimeout: param.PoolIdleTimeout,
		Wait:        true,
	}
	knownPool.Store(name, pool)
	return pool, nil
}
