package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Prepare() (*elasticsearch.Client, *gorm.DB, *zap.Logger) {

	logger, _ := zap.NewDevelopment()

	var k = koanf.New(".")
	if err := k.Load(file.Provider("../../config/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("加载配置失败 %v", err)
		return nil, nil, nil
	}

	db, err := NewTidbClient(k, logger)
	if err != nil {
		logger.Error("打开数据库失败", zap.Error(err))
		return nil, nil, nil
	}

	es, err := NewElasticClient(k)
	if err != nil {
		logger.Error("打开elastic失败", zap.Error(err))
		return nil, nil, nil
	}

	return es, db, logger
}

func initOrderIndex(dao *OrderDao) error {
	index := "trade_order"

	mapping := `
    {
      "settings": {
        "number_of_shards": 1
      },
      "mappings": {
        "properties": {
          "TotalAmount": {
            "type": "double"
          },
		  "DiscountAmount": {
            "type": "double"
          },
		  "PaymentAmount": {
            "type": "double"
          }
        }
      }
    }`

	res, err := dao.es.Indices.Create(index, dao.es.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func exportTidbToElastic(dao *OrderDao) error {

	index := "trade_order"

	_, orders, err := dao.getOrder(0, 100, 0, 0)

	if err != nil {
		return err
	}

	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{Index: index, Client: dao.es})

	if err != nil {
		return err
	}

	for _, to := range orders {

		str, err := json.Marshal(to)

		if err != nil {
			return err
		}

		indexer.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   bytes.NewReader(str),
			})

		// res, err := dao.es.Index(index, bytes.NewReader(str), dao.es.Index.WithDocumentID(to.TradeNo), dao.es.Index.WithRefresh("true"))

		// if err != nil {
		// 	return err
		// }

		// res.Body.Close()
	}

	err = indexer.Close(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func TestGetOrder(t *testing.T) {

	dao := NewOrderDao(Prepare())

	// err := initOrderIndex(dao)
	// if err != nil {
	// 	t.Errorf("初始化Index失败 %+v", err)
	// }

	// err := exportTidbToElastic(dao)

	// if err != nil {
	// 	t.Errorf("导入数据失败 %+v", err)
	// }

	// var tradeNo uint64 = 1536972017172901888
	// order, err := dao.GetOrder(0, 10, tradeNo, 0)
	// if err != nil {
	// 	logger.Error("查询订单失败", zap.Error(err))
	// }

	// logger.Info("查询结果", zap.Any("订单", order))

	total, order, err := dao.GetOrder(0, 10, 0, 0)
	if err != nil {
		dao.logger.Error("查询订单失败", zap.Error(err))
	}

	dao.logger.Info("查询结果", zap.Int64("总数", total), zap.Any("订单", len(order)))
}
