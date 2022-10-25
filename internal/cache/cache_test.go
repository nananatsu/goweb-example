package cache

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/zap"
)

func prepare() (*redis.Client, *zap.Logger) {

	logger, _ := zap.NewDevelopment()

	var k = koanf.New(".")
	if err := k.Load(file.Provider("../../config/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("加载配置失败 %v", err)
		return nil, nil
	}

	rdb := NewRedisClient(k)

	return rdb, logger
}

func TestPut(t *testing.T) {

	p := NewCache(prepare())

	if err := p.Put("test:test:1", "testKey", "testValue"); err != nil {
		t.Errorf("添加缓存失败 %v", err)
	}
}

func TestPutRange(t *testing.T) {

	p := NewCache(prepare())

	sortMembers := make([]redis.Z, 10)
	datamap := make(map[string]any)
	for i := 0; i < 10; i++ {
		i := i
		sortMembers[i] = redis.Z{Score: float64(i), Member: i}
		datamap[strconv.Itoa(i)] = i
	}

	if err := p.PutRange("test:sortset:2", "test:test:2", sortMembers, datamap, 10, 30*time.Second); err != nil {
		t.Errorf("添加缓存失败 %v", err)
	}

}

func TestGet(t *testing.T) {

	p := NewCache(prepare())

	data, err := p.Get("test:test:1", "testKey")

	if err != nil {
		t.Errorf("查询缓存失败 %v", err)
		return
	}

	if data != "testValue" {
		t.Error("查询缓存结果不一致")
	}

}

func TestGetRange(t *testing.T) {

	p := NewCache(prepare())

	total, data, err := p.Range("test:sortset:2", "test:test:2", 0, 10)

	if err != nil {
		t.Errorf("查询缓存失败 %v", err)
		return
	}

	t.Logf("查询结果: %+v , 总数: %d", data, total)

	if len(data) != 10 {
		t.Error("查询缓存结果不一致")
	}
}

// func TestGetRange2(t *testing.T) {

// 	p := NewCache(prepare())

// 	_, data, err := p.Range("sortset:order:0:0", "hashmap:order", 0, 10)

// 	if err != nil {
// 		t.Errorf("查询缓存失败 %v", err)
// 		return
// 	}

// 	if len(data) != 10 {
// 		t.Error("查询缓存结果不一致")
// 	}
// }
