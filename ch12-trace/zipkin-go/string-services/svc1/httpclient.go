//go:build go1.7
// +build go1.7

package svc1

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/openzipkin-contrib/zipkin-go-opentracing/examples/middleware"
)

// client is our actual client implementation
//封装了svc1的服务，例如执行client.Concat，则会通过http，向svc1微服务发送请求，
//拿到结果后返回执行的结果
type client struct {
	baseURL      string
	httpClient   *http.Client
	tracer       opentracing.Tracer
	traceRequest middleware.RequestFunc
}

// Concat implements our Service interface.
func (c *client) Concat(ctx context.Context, a, b string) (string, error) {
	// create new span using span found in context as parent (if none is found,
	// our span becomes the trace root).
	//如果ctx中有span，那么将ctx中的span作为父span，创建span，否则自己作为root span
	span, ctx := opentracing.StartSpanFromContext(ctx, "Concat")
	defer span.Finish()

	// assemble URL query
	url := fmt.Sprintf(
		"%s/concat/?a=%s&b=%s", c.baseURL, url.QueryEscape(a), url.QueryEscape(b),
	)

	// create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// use our middleware to propagate our trace
	//同样使用中间件，wrap一下http.request
	req = c.traceRequest(req.WithContext(ctx))

	// execute the HTTP request
	//将wrap之后的span发送出去
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// annotate our span with the error condition
		//如果出现异常情况，那么给span设置错误标签
		span.SetTag("error", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	// read the http response body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// annotate our span with the error condition
		span.SetTag("error", err.Error())
		return "", err
	}

	// return the result
	return string(data), nil
}

// Sum implements our Service interface.
func (c *client) Sum(ctx context.Context, a, b int64) (int64, error) {
	// create new span using span found in context as parent (if none is found,
	// our span becomes the trace root).
	//这里ctx是来自于http.Request的，所以如果客户端使用了zipkin库来wrap http请求，那么ctx中
	//是含有客户端span信息的。我们只需要做到operationName参数和客户端声明span的时候使用的是一致的，
	//即可将客户端的span作为root，创建我们自己的span，接着统计本trace
	span, ctx := opentracing.StartSpanFromContext(ctx, "Sum")
	defer span.Finish()

	// assemble URL query
	url := fmt.Sprintf("%s/sum/?a=%d&b=%d", c.baseURL, a, b)

	// create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// use our middleware to propagate our trace
	//使用zipkin库提供的中间件，wrap我们即将发送的http请求
	req = c.traceRequest(req.WithContext(ctx))

	// execute the HTTP request
	//执行http请求的发送。发送到下游的svc2。注意这里的req中，实际上含有了来自于客户端的span
	//信息，而每个span信息中又包含了traceID，spanID，parentID，所以可以最终构成完整的trace链路
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// annotate our span with the error condition
		span.SetTag("error", err.Error())
		return 0, err
	}
	defer resp.Body.Close()

	// read the http response body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// annotate our span with the error condition
		span.SetTag("error", err.Error())
		return 0, err
	}

	// convert html response to expected result type (int64)
	result, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		// annotate our span with the error condition
		span.SetTag("error", err.Error())
		return 0, err
	}

	// return the result
	return result, nil
}

// NewHTTPClient returns a new client instance to our svc1 using the HTTP
// transport.
//在cli中被调用，用于创建一个请求svc1服务的http客户端
func NewHTTPClient(tracer opentracing.Tracer, baseURL string) Service {
	return &client{
		baseURL:      baseURL,
		httpClient:   &http.Client{},
		tracer:       tracer,
		traceRequest: middleware.ToHTTPRequest(tracer),
	}
}
