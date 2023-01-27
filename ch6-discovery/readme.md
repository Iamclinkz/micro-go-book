本章借助consul，实现了一个服务发现微服务。

本目录本身是一个微服务，用http协议暴露具有say-hello，discovery，health三个功能的服务发现微服务。启动后将自己注册到consul中，并且借助consul，用以向消费者提供服务发现的能力。

整个框架从高层到底层，分别为：

```
transport层
endpoint层
service层
```

通过目录结构，总结golang微服务项目组织形式（以本章中的服务发现微服务为例）：

```sh
Exia➜  ch6-discovery : master ✘ :✹✚✭ ᐅ  tree .
.																			#顶层目录，存放服务发现微服务本身
├── config														#存放服务发现微服务的配置信息
│   └── config.go
├── discover													#（借助consul）实际的执行服务发现
│   ├── discover_client.go						#定义了提供服务发现的客户端需要满足的接口（服务注册、注销、发现）
│   ├── kit_discover_client.go				#使用go-kit实现的服务发现客户端
│   └── my_discover_client.go					#自己实现的服务发现客户端
├── endpoint													#endpoint类
│   └── endpoints.go
├── main.go														#用于启动本微服务
├── service														#service层
│   └── service.go										#定义了服务发现服务需要满足的接口，以及其实现类。实现类通过discover目│																		录下的discover_client.go的接口，（借助consul）实际的执行服务发现
└── transport													#transport层
    └── http.go												#将服务发现微服务通过http协议暴露出去
```

下面分层来细看每层的作用：

#### 1. service层

这层有点像http框架中的service层。

这层实际的执行服务发现的能力。因为是demo，所以直接通过discovery目录下的consul客户端，通过consul进行具体的服务发现。

##### 1.1 接口定义：

```go
type Service interface {

	// HealthCheck check service health status
	HealthCheck() bool

	// sayHelloService
	SayHello() string

	//  discovery service from consul by serviceName
	DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error)
}
```

##### 1.2 接口实现：

定义了一个实现类，用于实现这些service：

```go
//DiscoveryServiceImpl 利用内部的DiscoveryClient，实现了Service接口，即对外提供服务发现接口
type DiscoveryServiceImpl struct {
	//内部持有一个服务发现客户端（consul客户端）实例，对外提供服务发现服务。
	discoveryClient discover.DiscoveryClient
}
```

其他两个接口都是返回默认值，而DiscoveryService接口，借助内部持有的consul，实现服务发现：

```go
//DiscoveryService 传入serviceName，返回提供服务的实例
func (service *DiscoveryServiceImpl) DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error) {

	//直接使用了内部持有的discoveryClient的DiscoverServices方法
	instances := service.discoveryClient.DiscoverServices(serviceName, config.Logger)

	if instances == nil || len(instances) == 0 {
		return nil, ErrNotServiceInstances
	}
	return instances, nil
}
```

#### 2. endpoint层

这层功能相对比较简单，只是对service层的一个封装。每个endpoint相当于是某个微服务提供的一个“服务”，在transport层来说，应该将每个endpoint单独用于响应某个url的请求。在这一层，我们需要定义每个endpoint的请求和回复，类似于http框架中的reqeust和response。

例如服务发现的请求结构体和回复结构体为：

```go
// 服务发现请求结构体
type DiscoveryRequest struct {
	ServiceName string
}

// 服务发现响应结构体
type DiscoveryResponse struct {
	Instances []interface{} `json:"instances"`
	Error string `json:"error"`
}
```

这一层是借助了go-kit中的endpoint包中的Endpoint来实现的。查看位于：

`/Users/sunliyuan/sdk/go1.18/pkg/mod/github.com/go-kit/kit@v0.9.0/endpoint/endpoint.go`的源码：

```go
// Endpoint is the fundamental building block of servers and clients.
// It represents a single RPC method.
type Endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)
```

可以发现实际上每个Endpoint相当于是http框架中的一个controller。controller的作用是响应某个url的请求，拿到requset，进行处理，并返回response。Endpoint也是一样。

这层定义了用于响应三种请求的Endpoint的构造函数。结合前面的endpoint，可以看出，构造函数实际上返回的是一个endpoint.Endpoint类型的函数：

```go
// 创建服务发现的 Endpoint
func MakeDiscoveryEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(DiscoveryRequest)
		instances, err := svc.DiscoveryService(ctx, req.ServiceName)
		var errString = ""
		if err != nil{
			errString = err.Error()
		}
		return &DiscoveryResponse{
			Instances:instances,
			Error:errString,
		}, nil
	}
}
```

方便上层通过构造函数，创建endpoint。

##### 3. transport层

这层使用2.中提供的endpoint层中的endpoint，通过不同的协议，进行endpoint的输入输出的检查&编解码，然后交给endpoint使用。这层存在的意义是：

* 解耦编解码和endpoint。让endpoint不用关心发过来的请求是否不规范（不是业务层面的非法，而是例如json解析失败）
* 进行路由控制。在这一层将url和endpoint进行衔接，让一个endpoint处理一个url上坚挺的内容

这层的设计也很巧妙。接收endpoints之后，返回的是一个http.Handler。这样可以让上层处理网络连接管理，网络端口等内容。而不用下沉到transport这一层。这一层假设网络连接是好的，只是对网络进行管理：

```go
// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoints endpts.DiscoveryEndpoints, logger log.Logger) http.Handler {
	r := mux.NewRouter()

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
	}

	r.Methods("GET").Path("/say-hello").Handler(kithttp.NewServer(
		endpoints.SayHelloEndpoint,
		decodeSayHelloRequest,
		encodeJsonResponse,
		options...,
	))

	r.Methods("GET").Path("/discovery").Handler(kithttp.NewServer(
		endpoints.DiscoveryEndpoint,
		decodeDiscoveryRequest,
		encodeJsonResponse,
		options...,
	))

	// create health check handler
	r.Methods("GET").Path("/health").Handler(kithttp.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeJsonResponse,
		options...,
	))

	return r
}
```

