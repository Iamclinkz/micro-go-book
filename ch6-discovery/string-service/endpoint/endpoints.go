package endpoint

import (
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/longjoy/micro-go-book/ch10-resiliency/string-service/service"
	"strings"
)

// StringEndpoints define endpoint
//在这里定义下这个，说明清楚本目录下都有哪些endpoint
type StringEndpoints struct {
	StringEndpoint      endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

var (
	ErrInvalidRequestType = errors.New("RequestType has only two type: Concat, Diff")
)

// StringRequest define request struct
type StringRequest struct {
	//注意声明的时候可以加上tag，用来自定义json的解码
	RequestType string `json:"request_type"`
	A           string `json:"a"`
	B           string `json:"b"`
}

// StringResponse define response struct
type StringResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

// MakeStringEndpoint make endpoint
//构造StringEndpoint的构造函数，一个StringEndpoint可以理解成mvc框架中的一个"controller"，
//处理某个url上的请求，接收确定类型的request，调用对应的service，给出对应类型的response。
//增加这一层，是为了衔接外部url路径（而通常一个url表示一个对外的服务），和内部逻辑服务。即让内外服务解耦。
//所以这里需要传入一个 string-service.Service ，因为其本身不具有处理的能力。
func MakeStringEndpoint(svc service.Service) endpoint.Endpoint {
	//这里可以理解成service.Service是直接提供计算服务的服务，
	//而endpoint.Endpoint则是为微服务架构中的某个具体微服务本身的抽象。
	//通过endpoint.Endpoint，可以通过微服务框架（例如当前使用的go-kit）对微服务进行注册
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(StringRequest)

		var (
			res, a, b string
			opError   error
		)

		a = req.A
		b = req.B
		// 根据请求操作类型请求具体的操作方法
		if strings.EqualFold(req.RequestType, "Concat") {
			res, _ = svc.Concat(a, b)
		} else if strings.EqualFold(req.RequestType, "Diff") {
			res, _ = svc.Diff(a, b)
		} else {
			return nil, ErrInvalidRequestType
		}

		return StringResponse{Result: res, Error: opError}, nil
	}
}

// HealthRequest 健康检查请求结构
type HealthRequest struct{}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status bool `json:"status"`
}

// MakeHealthCheckEndpoint 创建健康检查Endpoint
func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthResponse{status}, nil
	}
}
