package transport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/longjoy/micro-go-book/ch6-discovery/string-service/endpoint"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	ErrorBadRequest = errors.New("invalid request parameter")
)

// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoints endpoint.StringEndpoints, logger log.Logger) http.Handler {
	//这里可以学一下kit的http包的使用
	//使用mux.NewRouter可以创建一个http handler接口的实例
	r := mux.NewRouter()

	//给kithttp.Server（即某个路径上的http.Handler）增加一个option
	options := []kithttp.ServerOption{
		//指定错误发生之后的handler
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		//如果编码出现错误，那么拿到编码的错误之后，应该怎么通过http.Response向请求者返回错误
		//如果不指定，默认使用DefaultErrorEncoder，可以看一下DefaultErrorEncoder的源码，
		//里面用的go的duck typing判断是否符合接口，很有意思
		kithttp.ServerErrorEncoder(encodeError),
	}

	//给http handler的对应url添加post方法(可以理解成指定一条路由的转发规则），符合此url的请求，会被转发给对应的kithttp.NewServer
	//其中kithttp.NewServer内部封装了一个endpoint.Endpoint,endpoint.Endpoint内部相当于封装了一个我们自定义的service
	//（这里虽然没有直接保存自定义service，但是调用了我们提供的service的方法）
	r.Methods("POST").Path("/op/{type}/{a}/{b}").Handler(kithttp.NewServer(
		//这里使用kithttp.NewServer，可以通过一个kit.Endpoint，创建一个kit.Server。
		//其中后者wrap了前者，后者实现了http.Handler接口，使用前者提供的service，通过http进行服务
		//这样可以解耦transport层和service层
		endpoints.StringEndpoint,
		//将request转换为上面的endpoints.StringEndpoint中的request的类型
		decodeStringRequest,
		//将上面的endpoints.StringEndpoint处理后生成的response类型，写入到http.ResponseWriter中
		encodeStringResponse,
		//另外附加的options
		options...,
	))

	r.Path("/metrics").Handler(promhttp.Handler())

	// create health check handler
	r.Methods("GET").Path("/health").Handler(kithttp.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeStringResponse,
		options...,
	))

	return r
}

// decodeStringRequest decode request params to struct
func decodeStringRequest(_ context.Context, r *http.Request) (interface{}, error) {
	//这里mux包将http request拿到之后，按照上面指定的"/op/{type}/{a}/{b}"这个格式解析之后，
	//将key-value（例如 "type"-"Diff","a"-"AAA")这样对放到request.ctx中。
	vars := mux.Vars(r)
	requestType, ok := vars["type"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pa, ok := vars["a"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pb, ok := vars["b"]
	if !ok {
		return nil, ErrorBadRequest
	}

	return endpoint.StringRequest{
		RequestType: requestType,
		A:           pa,
		B:           pb,
	}, nil
}

// encodeStringResponse encode response to return
func encodeStringResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// decodeHealthCheckRequest decode request
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoint.HealthRequest{}, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
