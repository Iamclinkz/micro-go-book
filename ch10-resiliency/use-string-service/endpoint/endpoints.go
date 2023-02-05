package endpoint

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"github.com/longjoy/micro-go-book/ch10-resiliency/use-string-service/service"
)

// CalculateEndpoint define endpoint
type UseStringEndpoints struct {
	UseStringEndpoint   endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

// StringRequest define request struct
type UseStringRequest struct {
	RequestType string `json:"request_type"`
	A           string `json:"a"`
	B           string `json:"b"`
}

// StringResponse define response struct
type UseStringResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

//// MakeStringEndpoint make endpoint
//func MakeUseStringEndpoint(svc service.Service) endpoint.Endpoint {
//	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
//		req := request.(UseStringRequest)
//
//		var (
//			res, a, b, opErrorString string
//			opError   error
//		)
//
//		a = req.A
//		b = req.B
//
//		res, opError = svc.UseStringService(req.RequestType, a, b)
//
//		if opError != nil{
//			opErrorString = opError.Error()
//		}
//
//		//如果是普通的，不在endpoint这一层使用hystrix的endpoint，这里error可以返回nil（即使服务返回错误）
//		//但是如果在endpoint这一层使用了hystrix，则如果服务出错，应该返回服务的错误，这样可以让hystrix感知到上游服务的
//		//错误，从而对断路器进行操作
//		return UseStringResponse{Result: res, Error: opErrorString}, nil
//	}
//}

func MakeUseStringEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UseStringRequest)

		var (
			res, a, b string
			opError   error
		)

		a = req.A
		b = req.B

		//这里的svc.UseStringService是没有容错处理的普通的http请求 service，如果http出现问题，则返回error
		res, opError = svc.UseStringService(req.RequestType, a, b)

		//将svc.UseStringService的错误返回，这样让层的hystrix如果看到错误，则会应用到断路器的统计中
		return UseStringResponse{Result: res}, opError
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
