package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"goweb/internal/cache"
	"goweb/internal/dao"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

type GetOrderParam struct {
	PageNumber int    `form:"pageNumber"`
	PageSize   int    `form:"pageSize"`
	TradeNo    uint64 `form:"tradeNo"`
	UserId     uint64 `form:"userId"`
}

type GetOrderResult struct {
	Total  int64             `json:"total"`
	Orders []*dao.TradeOrder `json:"orders"`
}

type OrderHandler struct {
	cache    *cache.Cache
	orderDao *dao.OrderDao
	logger   *zap.Logger
}

func (o *OrderHandler) GetHtml(c *gin.Context) {
	c.String(http.StatusOK, "hello")
}

func (o *OrderHandler) GetOrder(c *gin.Context) {

	var params GetOrderParam
	if err := c.ShouldBind(&params); err != nil {
		o.logger.Debug("参数解析异常", zap.Error(err))
		c.JSON(http.StatusBadRequest, Response[struct{}]{Code: http.StatusBadRequest, Message: "参数解析异常"})
		return
	}

	o.logger.Debug("解析查询参数", zap.Any("结果", params))

	offset := params.PageNumber * params.PageSize
	sortKey := fmt.Sprintf("sortset:order:%d:%d", params.UserId, params.TradeNo)
	dataKey := "hashmap:order"
	total, ret, err := o.cache.Range(sortKey, dataKey, int64(offset), int64((params.PageNumber+1)*params.PageSize)-1)

	if err != nil {
		o.logger.Warn("查询订单缓存失败", zap.Error(err))
	}

	var orders = make([]*dao.TradeOrder, params.PageSize)

	if len(ret) > 0 {
		for i, v := range ret {
			var order dao.TradeOrder

			err = json.Unmarshal([]byte(v), &order)
			if err != nil {
				o.logger.Warn("反序列化order失败", zap.Any("order", v), zap.Error(err))
				continue
			}
			orders[i] = &order
		}

	} else {
		total, orders, err = o.orderDao.GetOrder(params.PageNumber, params.PageSize, params.TradeNo, params.UserId)
		if err != nil {
			o.logger.Error("查询订单失败", zap.Error(err))
			c.JSON(http.StatusBadRequest, Response[struct{}]{Code: http.StatusBadRequest, Message: "查询订单失败"})
			return
		}

		var orderMap = make(map[string]any, len(orders))
		var members = make([]redis.Z, len(orders))
		for i, order := range orders {
			orderStr, err := json.Marshal(order)
			if err != nil {
				o.logger.Warn("序列化订单失败", zap.Error(err))
				continue
			}

			members[i] = redis.Z{Score: float64(offset + i), Member: order.TradeNo}
			orderMap[order.TradeNo] = orderStr
		}

		o.cache.PutRange(sortKey, dataKey, members, orderMap, total, time.Hour)
	}

	c.JSON(http.StatusOK, Response[*GetOrderResult]{Code: http.StatusOK, Data: &GetOrderResult{Total: total, Orders: orders}})

}

func (o *OrderHandler) AddOrder(c *gin.Context) {

}

func (o *OrderHandler) UpdateOrder(c *gin.Context) {

}

func (o *OrderHandler) DeleteOrder(c *gin.Context) {

}

func NewOrderHandler(cache *cache.Cache, orderDao *dao.OrderDao, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{cache: cache, orderDao: orderDao, logger: logger}
}
