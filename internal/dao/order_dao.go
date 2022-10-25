package dao

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/elastic/go-elasticsearch/v7"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EsResponse[T any] struct {
	Hits *EsHit[T] `json:"hits"`
}

type EsHit[T any] struct {
	Total *TotalHit      `json:"total"`
	Index []*IndexHit[T] `json:"hits"`
}

type TotalHit struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

type IndexHit[T any] struct {
	Index  string `json:"_index"`
	Id     string `json:"_id"`
	Source T      `json:"_source"`
}

type TradeOrder struct {
	TradeNo        string `gorm:"primaryKey"`
	UserId         string
	UserCode       string
	Nickname       string
	Subject        string
	TotalAmount    float64 `json:",string"`
	DiscountAmount float64 `json:",string"`
	PaymentAmount  float64 `json:",string"`
	ExpireTime     *LocalTime
	TradeStatus    int
	CreateTime     *LocalTime
	CreateUser     string
	UpdateTime     *LocalTime
	UpdateUser     string
	Deleted        int
}

type OrderDao struct {
	es     *elasticsearch.Client
	db     *gorm.DB
	logger *zap.Logger
}

func (dao *OrderDao) GetOrder(page, size int, tradeNo, userId uint64) (int64, []*TradeOrder, error) {
	offset := page * size
	matchMap := make(map[string]interface{})
	if tradeNo != 0 {
		matchMap["trade_no"] = tradeNo
	}

	if userId != 0 {
		matchMap["user_id"] = userId
	}

	var buf bytes.Buffer
	query := map[string]interface{}{
		"from": offset,
		"size": size,
	}
	if len(matchMap) > 0 {
		query["query"] = map[string]interface{}{
			"match": matchMap,
		}
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		dao.logger.Error("序列化请求条件失败", zap.Error(err))
		return 0, nil, err
	}

	res, err := dao.es.Search(
		dao.es.Search.WithContext(context.Background()),
		dao.es.Search.WithIndex("trade_order"),
		dao.es.Search.WithBody(&buf),
		dao.es.Search.WithTrackTotalHits(true),
		dao.es.Search.WithPretty(),
	)

	if err != nil {
		dao.logger.Error("查询elastic失败", zap.Error(err))
		return 0, nil, err
	}

	var ret EsResponse[*TradeOrder]
	err = json.NewDecoder(res.Body).Decode(&ret)

	if err != nil {
		dao.logger.Error("反序列化查询结果失败", zap.Error(err))
		return 0, nil, err
	}

	orders := make([]*TradeOrder, len(ret.Hits.Index))
	for i, ih := range ret.Hits.Index {
		orders[i] = ih.Source
	}

	return ret.Hits.Total.Value, orders, nil

}

func (dao *OrderDao) getOrder(page, size int, tradeNo, userId uint64) (int64, []*TradeOrder, error) {

	var order []*TradeOrder
	var count int64

	var db = dao.db.
		Model(&order).
		Select("trade_no, trade_order.user_id, tus.user_code, tus.nickname, subject, total_amount, discount_amount, payment_amount, expire_time, trade_status, create_time, create_user, update_time, update_user, is_deleted").
		Joins("LEFT JOIN trade_user_sync tus on tus.user_id = trade_order.user_id").
		Where("trade_order.is_deleted = 0")

	if tradeNo > 0 {
		db = db.Where("trade_order.trade_no = ?", tradeNo)
	}
	if userId > 0 {
		db = db.Where("trade_order.user_id = ?", userId)
	}

	ret := db.Order("trade_order.trade_no desc").
		Offset(page * size).
		Limit(size).
		Find(&order).
		Offset(-1).
		Limit(1).
		Count(&count)

	if ret.Error != nil {
		return 0, nil, ret.Error
	}

	dao.logger.Debug("查询数据数量", zap.Int64("count", count))

	return count, order, nil
}

func NewOrderDao(es *elasticsearch.Client, db *gorm.DB, logger *zap.Logger) *OrderDao {
	return &OrderDao{es: es, db: db, logger: logger}
}

func ProvideOrderDao() fx.Option {
	return fx.Provide(NewElasticClient, NewTidbClient, NewOrderDao)
}
