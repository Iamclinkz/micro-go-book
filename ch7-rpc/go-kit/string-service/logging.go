package string_service

import (
	"context"
	"github.com/go-kit/kit/log"
	"time"
)

// loggingMiddleware Make a new type
// that contains Service interface and logger instance
//利用了golang的存储接口的特性。由于loggingMiddleware内部持有了一个实现了string-service.Service接口的实例，所以
//其本身也实现了string-service.Service接口。这个特性适合做装饰器模式
type loggingMiddleware struct {
	Service
	logger log.Logger
}

// LoggingMiddleware make logging middleware
func LoggingMiddleware(logger log.Logger) ServiceMiddleware {
	//注意这种声明中间件的方式，调用者调用LoggingMiddleware（logger），拿到的实际上是一个函数
	//这个函数原型为：
	//func(next Service)Service，即如果希望使用这个函数，需要再传入一个service，有点像洋葱，
	//传入洋葱内层，包裹本层皮，再返回
	return func(next Service) Service {
		return loggingMiddleware{next, logger}
	}
}

func (mw loggingMiddleware) Concat(ctx context.Context, a, b string) (ret string, err error) {

	//利用适配器模式，给每个wrap的服务，增加了一个日志记录
	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "Concat",
			"a", a,
			"b", b,
			"result", ret,
			"took", time.Since(begin),
		)
	}(time.Now())

	//这里实际上是wrapper模式，实际执行Concat的还是内部持有的Service
	ret, err = mw.Service.Concat(ctx, a, b)
	return ret, err
}

func (mw loggingMiddleware) Diff(ctx context.Context, a, b string) (ret string, err error) {

	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "Diff",
			"a", a,
			"b", b,
			"result", ret,
			"took", time.Since(begin),
		)
	}(time.Now())

	ret, err = mw.Service.Diff(ctx, a, b)
	return ret, err
}

func (mw loggingMiddleware) HealthCheck() (result bool) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "HealthChcek",
			"result", result,
			"took", time.Since(begin),
		)
	}(time.Now())
	result = true
	return
}
