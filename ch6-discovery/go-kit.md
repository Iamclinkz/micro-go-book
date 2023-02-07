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
//这样定义有个好处，就是直接使用包名 + Service，例如 discovery.Service 即可使用discovery的Service
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

#### 3. transport层

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

  //这里注意，kit中的Server同Endpoint是1:1的关系
  //Server = Endpoint（aka controller） + decode函数 + encode函数
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

#### 4.service和endpoint的middleware

service层和endpoint层都可以声明自己的middleware，用于面向切面编程。middleware通常是以函数的形式实现。

每个Endpoint实际上是一个`func`，而每个Service实际上是一个`interface`，这两个的区别是，前者只能做handler，无法自己持有中间变量（只能可以通过上层传入func），相当于是无状态的。而后者由于是`interface`，所以一定是struct实现的，而struct内部是可以持有自己的变量的。

* ServiceMiddleware

  service层的`ServiceMiddleware`应该是对Service的封装，即传入一个提供服务功能的Service，传出一个加了中间件后，继续提供服务功能的Service（就好像传入一个士兵，士兵拥有“打仗”这个功能，在ServiceMiddleware中，给士兵加了“打仗前喊口号”这个行为，再传出。士兵仍然只是暴露“打仗”这个 功能）：

  ```go
  //自己生命的ServiceMiddleware的类型
  type ServiceMiddleware func(Service) Service
  ```

  假设我们希望通过构造函数创建不同类型的Middleware，那么因为service是一个自定义的接口类型，所以作为参数传入的Service应该是struct类型。如果我们希望wrap一下struct，需要另外声明一个struct，用新的struct来保存作为参数的struct。所以这里的具体操作，应该是重新声明一个struct类型， 然后将传入的Service作为其成员变量，然后返回新的struct类型。举例来说，如果我们希望加一个日志中间件，应该如此定义：

  ```go
  //contains Service interface and logger instance
  //利用了golang的存储接口的特性。由于loggingMiddleware内部持有了一个实现了string-service.Service接口的实例，所以其本身也实现了string-service.Service接口。这个特性适合做装饰器模式
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
  ```

  这样，我们通过调用：

  ```go
  var svc service.Service
  svc = service.StringService{}
  
  // add logging middleware
  svc = service.LoggingMiddleware(logger)(svc)
  ```

  即可实现对service添加中间件。

* EndpointMiddleware

  endpoint层的`Middleware`应该是对Endpoint的封装。这部分是框架支持的：

  ```go
  //位于/Users/sunliyuan/sdk/go1.18/pkg/mod/github.com/go-kit/kit@v0.9.0/endpoint/endpoint.go
  type Middleware func(endpoint.Endpoint) endpoint.Endpoint
  ```

  由于endpoint.Endpoint的类型为：

  ```go
  type Endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)
  ```

  所以，中间件的构造函数的例子为：

  ```go
  func NewEndpointLogMiddlewareExample(logger log.Logger) endpoint.Middleware {
    //通过调用下面一行return的函数，就可以拿到洋葱包裹器
  	return func(next endpoint.Endpoint) endpoint.Endpoint {
      //同serivceMiddleware，传入洋葱内层，包裹本层皮，再返回。这里返回的相当于是加了本层洋葱外壳的洋葱
  		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
        //执行本层操作。实际上logger也是外层传入的。但是本层并没有持有任何的自己的内容
  			if err = logger.Log("current handle:%v", ctx.Value("test")); err != nil {
  				return nil, err
  			}
        
        //本层处理完毕，让下一层继续处理。
  			return next(ctx, request)
  		}
  	}
  }
  ```

  