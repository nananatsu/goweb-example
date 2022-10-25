package di

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"goweb/internal/build"

	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitConf() (*koanf.Koanf, error) {
	var k = koanf.New(".")
	if err := k.Load(file.Provider("config/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("加载配置失败 %v", err)
		return nil, err
	}
	return k, nil
}

func NewLogWriter(k *koanf.Koanf) io.Writer {
	f, err := os.OpenFile(k.String("log.file"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		fmt.Printf("创建日志文件失败: %v", err)
	}

	writer := io.MultiWriter(f, os.Stdout)

	gin.DefaultWriter = writer

	return writer
}

func NewLogger(writer io.Writer) *zap.Logger {

	encoderConfig := zap.NewProductionEncoderConfig()
	timeFormat := "2006/01/02 15:04:05.000"

	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		type appendTimeEncoder interface {
			AppendTimeLayout(time.Time, string)
		}

		if enc, ok := enc.(appendTimeEncoder); ok {
			enc.AppendTimeLayout(t, timeFormat)
			return
		}
		enc.AppendString(t.Format(timeFormat))
	}

	var encoder zapcore.Encoder
	if build.Debug {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	logWriter := zapcore.AddSync(writer)

	core := zapcore.NewCore(encoder, logWriter, zapcore.DebugLevel)

	return zap.New(core, zap.AddCaller())
}

func NewServer(k *koanf.Koanf, router *gin.Engine, lc fx.Lifecycle, logger *zap.Logger) *http.Server {

	srv := &http.Server{Addr: fmt.Sprintf("%s:%d", k.String("server.host"), k.Int("server.port")), Handler: router}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			logger.Info("启动http服务器", zap.String("address", srv.Addr))
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func ProvideConfig() fx.Option {
	return fx.Provide(InitConf)
}

func ProvideLogger() fx.Option {
	return fx.Provide(NewLogWriter, NewLogger)
}

func ProvideServer() fx.Option {
	return fx.Provide(NewServer)
}
