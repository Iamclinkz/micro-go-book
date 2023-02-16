# 1.项目结构

```sh
string-services								#顶层项目
├── Makefile									#整个项目的Makefile文件，用于构建本项目中的可执行文件
├── README.md
├── build											#用于存放本项目中的可执行文件，以及编译过程中的内容
│   ├── cli										#cli/main.go编译的内容
│   ├── svc1									#svc1/cmd/main.go编译的内容
│   └── svc2									#svc2/cmd/main.go编译的内容
├── cli												#控制台客户端，用于和其他模块通信
│   └── main.go
├── svc1											#微服务1
│   ├── cmd										#下面存放main函数，在此目录下执行go build，即可编译生成可执行文件
│   │   └── main.go						#将service的内容和controller的内容整合，并且决定使用何种方式提供服务
│   ├── httpclient.go					#用于和本微服务通信的http客户端
│   ├── httpserver.go					#本微服务的http的handler。存放controller（某个路径的controller，以及总的controller）
│   ├── implementation.go			#service.go中接口的实现
│   └── service.go						#本微服务的核心服务（service层）的接口定义，用于在controller层中调用
└── svc2											#微服务2。下略
    ├── cmd
    │   └── main.go
    ├── httpclient.go
    ├── httpserver.go
    ├── implementation.go
    └── service.go
```

# 2.集成zipkin流程

### 2.0 zipkin架构概览：

From:https://zipkin.io/pages/architecture.html

![Zipkin architecture](https://zipkin.io/public/img/architecture-1.png)

### 2.1 zipkin基本概念

##### 2.1.1 Trace

表示一次完整的调用链路。每个Trace表示某个事务或者请求在整个的系统中执行的过程。由多个span构成。使用TraceID来表示唯一性。每个Trace中的Span，可以构成一个有向无环图，即每个span（除了root外）都拥有一个父亲，以及0～n个孩子。

##### 2.1.2 Span

最基本的**工作**单元（即实际在链路中编码传输的单元），每个Span可以用于表示一次链路的调用，例如rpc调用，http调用等。主要用于描绘这一次调用的时间（还可以加一些别的tag，表示本次调用过程中的其他信息）。使用spanID来表示某个span在某个trace中的唯一性。除此之外，每个span中应该还包括：

* 描述信息：可以用于记录本次span执行过程中的信息

* 时间戳

* 键值对的tag信息（Annotation）：用于记录特定事件的相关信息，通常，约定俗成的在span中增加这四个Annotation：

  * CS：client send，客户端发起请求
  * SR：server receive，服务器收到请求
  * SS： server send，服务器处理结束，将结果返回给客户端
  * CR：client receive，客户端收到服务器返回的结果

  关于这四个约定俗称的Annotaiton的赋值时机，举例From:https://blog.csdn.net/THMAIL/article/details/96429436：

  > 假设有：**testService(Web服务) -> OrderServ(Thrift) -> StockServ & PayServ(Thrift)。**
  >
  > - testService收到Http Reqeust时，需在入口处生成TraceID、SpanID，以及一个Span对象，假若叫Span1。
  > - testService向OrderServ发送 Thrift Request时，需新生成一Span2，并把parent ID设置成Span1的spanID。同事需修改Thrift Header，把Span2的spanID、parent ID、TraceID 传递给下游服务。也需生成"cs" Annotation，关联到span2上；当接受到OrderServ的Response时，再生成"cr" Annotation，也关联到span2上。
  > - OrderServ接受到Thrift Request后，从Thrift Header里解析到TraceID、parent ID、 Span ID(span2)、并保留到上下文里。同时生成"sr"Annotaition，并关联到span2上；当处理完成发送response时，再生成"ss"Annotation，并关联到span2上。
  > - OrderServ向StockServ发送 Thrift Request时，需新生成一Span3，并把parentID设置成上一步(Span2)的span ID。Annotation处理如上。
  > - Order Serv向PayServ发送请求时，新生成一Span4，并把parentID设置Span2的span ID。Annotation处理如上

* 用于表示父Span的ParentID

* TraceID

>  注意需要和trace区分，一次trace（请求）可以包括若干span（调用）。

注意，通过SpanID，TraceID，ParentID这三个字段的传递，在zigkin server侧进行聚类和整合，可以描绘出某一次请求在整个系统中完整的传递，执行的拓扑图。而其他字段，例如Annotation的传递，可以用于携带某个span的更多数据，最终可以用于分析某个span的执行过程等。

### 2.2 go使用zipkin客户端

##### 2.2.0 zipkin server搭建

使用docker：

```sh
docker run -d -p 9411:9411 openzipkin/zipkin
```

启动zipkin，启动之后可以通过：

```sh
localhost:9411/zipkin
```

查看zipkin的ui界面。

这样搭建起来的zipkin实际上是一个集齐Collector 收集器、Storage 存储、API、UI 用户界面的 Zipkin Server 部分。

##### 2.2.1 客户端侧

0. 初始化本微服务的zipkin配置：

   ```go
   // collector->recorder->tracer
   // Create our HTTP collector.
   
   //	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"
   collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
   if err != nil {
     fmt.Printf("unable to create Zipkin HTTP collector: %+v\n", err)
     os.Exit(-1)
   }
   
   // Create our recorder.
   //debug指定是否使用debug模式
   //hostPort指定我们服务的host+port
   //serviceName指定我们服务的名称
   recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
   
   // Create our tracer.
   tracer, err := zipkin.NewTracer(
     recorder,
     zipkin.ClientServerSameSpan(sameSpan),		//rpc style span
     zipkin.TraceID128Bit(traceID128Bit),			//traceID应该生成128位
   )
   
   // 将我们的tracer设置为全局tracer
   opentracing.InitGlobalTracer(tracer)
   ```

1. 使用我们的tracer,初始化一个用于wrap http请求的中间件：

   ```go
   traceRequest := middleware.ToHTTPRequest(tracer),
   ```

2. 创建一个root span：

    ```go
    // Create Root Span for duration of the interaction with svc1
    span := opentracing.StartSpan("Run")
    ```

3. 使用该root span，创建ctx：

    ```go
    ctx := opentracing.ContextWithSpan(context.Background(), span)
    ```

4. 在发送http请求，请求另一个服务之前，将此ctx解析回span，并且提前设置span的结束时间为发送结束：

    ```go
    span, ctx := opentracing.StartSpanFromContext(ctx, "Concat")
    defer span.Finish()
    ```

5. 使用go库，创建一个http请求：

    ```go
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
      return "", err
    }
    ```

6. 使用刚刚用tracer初始化的中间件，wrap一下request，生成一个新的request：

    ```go
    req = c.traceRequest(req.WithContext(ctx))
    ```

7. 将请求发送，并且如果返回的内容有问题，通过设置一个spanTag来表明本次调用失败：

    ```go
    resp, err := c.httpClient.Do(req)
    if err != nil {
      // annotate our span with the error condition
      span.SetTag("error", err.Error())
      return 0, err
    }
    ```

##### 2.2.2 服务器侧

###### 2.2.2.1 http span

0. 初始化服务器侧的zipkin设置。同客户端侧相同，略。

1. 使用tracer，初始化http.Handler，使用这个handler来监听带有span的http请求：

   ```go
   //需要利用初始化好的zipkin.tracer，初始化一个zipkin中间件，然后再使用这个中间件，wrap我们的handler
   //这样如果接收到同样使用了zipkin wrap的http请求，就可以解析出来自于客户端的zipkin span，并且使用客户端
   //发来的zipkin span作为root，继续trace
   concatHandler = middleware.FromHTTPRequest(tracer, "操作的名称")(httpHandler)
   ```

2. 拿到http请求之后，从request中的ctx中解析出span，并且可以对span操作（例如加tag等）：

    ```go
    span := opentracing.SpanFromContext(req.Context())
    //对span进行操作。。。
    ```

3. 无需别的操作，将结果让handler帮忙返回即可。

###### 2.2.2.2 自定义span

```go
resourceSpan := opentracing.StartSpan(
  //因为不是上游ctx解析出来的，所以这里需要自己定义span的名称，以及是哪个span的孩子
 	//从span.Context()可以拿到span所属于的trace的traceID，以及整个的trace路径。
  "myComplexQuery",
  opentracing.ChildOf(span.Context()),
)
defer resourceSpan.Finish()
// mark span as resource type
//标识span的名称为resource
ext.SpanKind.Set(resourceSpan, "resource")
// name of the resource we try to reach
ext.PeerService.Set(resourceSpan, "PostgreSQL")
// hostname of the resource
ext.PeerHostname.Set(resourceSpan, "localhost")
// port of the resource
ext.PeerPort.Set(resourceSpan, 3306)
// let's binary annotate the query we run
resourceSpan.SetTag(
  "query", "SELECT recipes FROM cookbook WHERE topic = 'world domination'",
)
```

最终的结果如图：

![image-20230213002538735](/Users/sunliyuan/Library/Application Support/typora-user-images/image-20230213002538735.png)

注意，因为上层没有像http发送request请求一样，生成span，然后填入client send这个annotation，然后再发给下游微服务，下游微服务收到之后再从中取出span，填写server receive。即本微服务实际上没有下游微服务填写server receive和server send。所以本span的annotation只有client开始和client结束两个字段。

##### 2.2.3 zipkin server侧

每当某个span结束，zipkin客户端的（是recorder还是collector?)都会将该span的信息上报给zipkin server的collector，汇报的信息即zipkin的基本信息（包括traceID，spanID，span parent），以及用户附加信息（例如annotation，tag），当一个trace过程中的每个span，都发给服务器，并且服务器接收汇总之后，就可以绘制出整个的一个trace链路，以及这个链路中每个span的信息。

### 2.3 结果展示

通过ui界面（http://localhost:9411/zipkin）查看demo中的trace：

![image-20230212231026600_副本](/Users/sunliyuan/Downloads/image-20230212231026600_副本.png)

其中除了root span（因为root span不是server，没有承接其他span的请求），都会通过zipkin的go包，自动添加以下Annotation：

<img src="/Users/sunliyuan/Library/Application Support/typora-user-images/image-20230212232244621.png" alt="image-20230212232244621" style="zoom:50%;" />

从上到下，依次是本服务的上游开始请求时间，本服务接收到上游请求时间，本服务处理请求结束时间，上游收到本服务回复时间。