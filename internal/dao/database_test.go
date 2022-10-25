package dao

import (
	"fmt"
	"testing"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/zap"
)

func prepare() (*koanf.Koanf, *zap.Logger, error) {

	var k = koanf.New(".")
	if err := k.Load(file.Provider("../../config/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("加载配置失败 %v", err)
		return nil, nil, err
	}

	logger, err := zap.NewDevelopment()

	if err != nil {
		fmt.Printf("创建日志失败 %v", err)
		return nil, nil, err
	}

	return k, logger, nil
}

func TestConnect(t *testing.T) {

	k, _, err := prepare()

	if err != nil {
		t.Error(err)
	}

	c, err := NewElasticClient(k)

	if err != nil {
		t.Error(err)
	}

	t.Log(c.Info())
}
