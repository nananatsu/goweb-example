package handler

import (
	"encoding/json"
	"fmt"
	"goweb/internal/cache"
	"goweb/internal/dao"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/fx"
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

func di() []fx.Option {
	return []fx.Option{fx.Provide(prepare), dao.ProvideOrderDao(), cache.ProvideCache(), ProvideRouter()}
}

func TestGetOrder(t *testing.T) {

	ops := di()
	done := make(chan struct{})

	ops = append(ops, fx.Invoke(func(r *gin.Engine) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/order?pageNumber=0&pageSize=10", nil)

		r.ServeHTTP(w, req)

		var res Response[*GetOrderResult]
		json.NewDecoder(w.Result().Body).Decode(&res)

		if w.Code != http.StatusOK {
			t.Error("查询请求响应不为200", w.Code)
		}

		if len(res.Data.Orders) != 10 {
			t.Error("查询结果数量不为10", len(res.Data.Orders))
		}

		done <- struct{}{}
	}))

	go func() {
		fx.New(ops...).Run()
	}()

	<-done

}
