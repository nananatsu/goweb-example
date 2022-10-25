package cache

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/knadh/koanf"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const expireKeySortSet = "expire_key_sort_set"
const totalHitMap = "total_hit_map"

var rangeScript = redis.NewScript(`local k = redis.call('ZRANGE',KEYS[1],ARGV[1],ARGV[2]) 
if (#k > 0) then 
    redis.call('ZINCRBY',KEYS[3],3600,ARGV[3])
    return {redis.call('HMGET',KEYS[4],KEYS[1]),redis.call('HMGET',KEYS[2],unpack(k))}
else
    return {0,k}
end`)

type CacheLocation struct {
	SortSet string
	Hashmap string
}

type Cache struct {
	lv1Cache any
	lv2Cache *redis.Client
	logger   *zap.Logger
}

func (c *Cache) clearExpireKey() {

	ticker := time.NewTicker(30 * time.Second)
	for {
		t := <-ticker.C

		now := t.Unix()
		cmd := c.lv2Cache.ZRange(expireKeySortSet, 0, now)
		if cmd.Err() != nil {
			c.logger.Warn("获取缓存失败", zap.String("key", expireKeySortSet), zap.Int64("expireTime", now), zap.Error(cmd.Err()))
			continue
		}

		for _, member := range cmd.Val() {
			var cacheInfo CacheLocation
			err := json.Unmarshal([]byte(member), &cacheInfo)
			if err != nil {
				c.logger.Warn("解析缓存失败", zap.String("member", member), zap.Error(cmd.Err()))
				continue
			}

			var cur uint64
			var keys []string
			for {
				cmd := c.lv2Cache.ZScan(cacheInfo.SortSet, cur, "*", 100)
				if cmd.Err() != nil {
					c.logger.Warn("获取缓存失败", zap.String("key", cacheInfo.SortSet), zap.Uint64("current", cur), zap.Error(cmd.Err()))
					continue
				}

				keys, cur = cmd.Val()
				count := len(keys)

				if count > 0 {
					c.lv2Cache.HDel(cacheInfo.Hashmap, keys...)
				}
				if count < 100 {
					break
				}
			}

			icmd := c.lv2Cache.HDel(totalHitMap, cacheInfo.SortSet)
			if icmd.Err() != nil {
				c.logger.Warn("删除缓存失败", zap.String("key", totalHitMap), zap.String("filed", cacheInfo.SortSet), zap.Error(icmd.Err()))
				continue
			}

			cmd := c.lv2Cache.Del(cacheInfo.SortSet)
			if cmd.Err() != nil {
				c.logger.Warn("删除缓存失败", zap.String("key", cacheInfo.SortSet), zap.Error(cmd.Err()))
				continue
			}

			cmd = c.lv2Cache.ZRem(expireKeySortSet, member)
			if cmd.Err() != nil {
				c.logger.Warn("删除缓存失败", zap.String("key", expireKeySortSet), zap.String("member", member), zap.Error(cmd.Err()))
				continue
			}
		}
	}
}

func (c *Cache) Put(key, field string, value any) error {
	return c.lv2Cache.HSet(key, field, value).Err()
}

func (c *Cache) PutRange(sortKey, dataKey string, sort []redis.Z, data map[string]any, total int64, expire time.Duration) error {

	cmds, err := c.lv2Cache.TxPipelined(func(p redis.Pipeliner) error {
		if err := p.ZAdd(sortKey, sort...).Err(); err != nil {
			return err
		}

		member, err := json.Marshal(CacheLocation{SortSet: sortKey, Hashmap: dataKey})
		if err != nil {
			return err
		}

		if err := p.ZAdd(expireKeySortSet, redis.Z{Score: float64(time.Now().Add(expire).Unix()), Member: member}).Err(); err != nil {
			return err
		}

		if err := p.HMSet(dataKey, data).Err(); err != nil {
			return err
		}

		if err := p.HSet(totalHitMap, sortKey, total).Err(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		if cmd.Err() != nil {
			return err
		}
	}

	return nil
}

func (c *Cache) Get(key, field string) (any, error) {

	cmd := c.lv2Cache.HGet(key, field)

	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	return cmd.Val(), nil
}

func (c *Cache) Range(sortKey, dataKey string, start, end int64) (int64, []string, error) {

	expireKey, err := json.Marshal(CacheLocation{SortSet: sortKey, Hashmap: dataKey})
	if err != nil {
		return 0, nil, err
	}

	cmd := rangeScript.Run(c.lv2Cache, []string{sortKey, dataKey, expireKeySortSet, totalHitMap}, start, end, expireKey)

	if cmd.Err() != nil {
		return 0, nil, cmd.Err()
	}

	ret, ok := cmd.Val().([]any)

	var total int64
	var record []string

	if ok && len(ret) == 2 {
		ret1, ok1 := ret[0].([]any)
		ret2, ok2 := ret[1].([]any)
		if ok1 && ok2 {
			if len(ret1) == 1 {
				totalStr, _ := ret1[0].(string)
				total, err = strconv.ParseInt(totalStr, 10, 0)
				if err != nil {
					return total, record, err
				}

				record = make([]string, len(ret2))
				for i := range ret2 {
					str, _ := ret2[i].(string)
					record[i] = str
				}
			}
		}
	}
	return total, record, err
}

func NewCache(remoteCache *redis.Client, logger *zap.Logger) *Cache {
	p := &Cache{
		lv1Cache: nil,
		lv2Cache: remoteCache,
		logger:   logger,
	}

	go p.clearExpireKey()

	return p
}

func NewRedisClient(k *koanf.Koanf) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", k.String("db.redis.host"), k.Int("db.redis.port")),
		Password: k.String("db.redis.password"),
		DB:       k.Int("db.redis.db"),
	})
}

func ProvideCache() fx.Option {
	return fx.Provide(NewRedisClient, NewCache)
}
