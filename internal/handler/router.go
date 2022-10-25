package handler

import (
	"fmt"
	"net/http"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewRouter(orderHandler *OrderHandler, logger *zap.Logger) *gin.Engine {
	r := gin.New()

	r.Use(ginzap.Ginzap(logger, "2006/01/02 15:04:05.000", true))

	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.Error("请求异常", zap.Any("error", recovered))
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	order := r.Group("/order")
	{
		order.GET("", orderHandler.GetOrder)
		order.POST("/", orderHandler.AddOrder)
		order.PUT("/", orderHandler.UpdateOrder)
		order.DELETE("/", orderHandler.DeleteOrder)
	}

	return r
}

func ProvideRouter() fx.Option {
	return fx.Provide(NewOrderHandler, NewRouter)
}
