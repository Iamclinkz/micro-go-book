//go:build go1.7
// +build go1.7

package svc1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/opentracing/opentracing-go"

	"github.com/openzipkin-contrib/zipkin-go-opentracing/examples/middleware"
)

//httpService 用于controller和service解耦，在controller这一层，只能通过接口来使用service的方法。
//同样，service应该同controller划清界限，即只应该暴露Service接口中的方法。
type httpService struct {
	service Service
}

// concatHandler is our HTTP HandlerFunc for a Concat request.
func (s *httpService) concatHandler(w http.ResponseWriter, req *http.Request) {
	// parse query parameters
	v := req.URL.Query()
	result, err := s.service.Concat(req.Context(), v.Get("a"), v.Get("b"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// return the result
	w.Write([]byte(result))
}

// sumHandler is our HTTP Handlerfunc for a Sum request.
func (s *httpService) sumHandler(w http.ResponseWriter, req *http.Request) {
	// parse query parameters
	v := req.URL.Query()
	a, err := strconv.ParseInt(v.Get("a"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := strconv.ParseInt(v.Get("b"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// call our Sum binding
	//这里从http.Request中拿出了ctx，然后传给了Service层。而如果来自客户端的
	//http请求中，含有zipkin的span数据，Service层即可以配合客户端的span数据，进行自己的统计，并通过
	//http.Response返回给客户端
	result, err := s.service.Sum(req.Context(), a, b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// return the result
	w.Write([]byte(fmt.Sprintf("%d", result)))
}

// NewHTTPHandler returns a new HTTP handler our svc1.
func NewHTTPHandler(tracer opentracing.Tracer, service Service) http.Handler {
	// Create our HTTP Service.
	svc := &httpService{service: service}

	// Create the mux.
	mux := http.NewServeMux()

	// Create the Concat handler.
	//传入一个service，创建一个http handler
	var concatHandler http.Handler
	concatHandler = http.HandlerFunc(svc.concatHandler)

	// Wrap the Concat handler with our tracing middleware.
	//需要利用初始化好的zipkin.tracer，初始化一个zipkin中间件，然后再使用这个中间件，wrap我们的handler
	//这样如果接收到同样使用了zipkin wrap的http请求，就可以解析出来自于客户端的zipkin span，并且使用客户端
	//发来的zipkin span作为parent，继续trace
	concatHandler = middleware.FromHTTPRequest(tracer, "Concat")(concatHandler)

	// Create the Sum handler.
	var sumHandler http.Handler
	sumHandler = http.HandlerFunc(svc.sumHandler)

	// Wrap the Sum handler with our tracing middleware.
	//同样的方式操作Sum请求的handler
	sumHandler = middleware.FromHTTPRequest(tracer, "Sum")(sumHandler)

	// Wire up the mux.
	mux.Handle("/concat/", concatHandler)
	mux.Handle("/sum/", sumHandler)

	// Return the mux.
	return mux
}
