package dao

import (
	"fmt"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/estransport"
	"github.com/knadh/koanf"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

func NewElasticClient(k *koanf.Koanf) (*elasticsearch.Client, error) {

	cfg := elasticsearch.Config{
		Addresses: strings.Split(k.String("db.es.address"), ","),
		Username:  k.String("db.es.user"),
		Password:  k.String("db.es.password"),
		Logger:    &estransport.ColorLogger{Output: os.Stdout},
	}
	return elasticsearch.NewClient(cfg)
}

func NewTidbClient(k *koanf.Koanf, lg *zap.Logger) (*gorm.DB, error) {

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4", k.String("db.tidb.user"), k.String("db.tidb.password"),
		k.String("db.tidb.host"), k.Int("db.tidb.port"), k.String("db.tidb.db"))
	gormLg := zapgorm2.New(lg)
	gormLg.SetAsDefault()
	gormLg.LogLevel = logger.Info

	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: gormLg,
	})
}
