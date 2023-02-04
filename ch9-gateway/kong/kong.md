## 1. kong的启动

### 1.1 启动一个不带数据库的kong

From：https://docs.konghq.com/gateway/latest/install/docker/?_ga=2.221067996.519989112.1675267286-1380262659.1675267286#start-kong-gateway-in-db-less-mode

##### step1.export一下kong的license:

```bash
 export KONG_LICENSE_DATA='{"license":{"payload":{"admin_seats":"1","customer":"Example Company, Inc","dataplanes":"1","license_creation_date":"2017-07-20","license_expiration_date":"2017-07-20","license_key":"00141000017ODj3AAG_a1V41000004wT0OEAU","product_subscription":"Konnect Enterprise","support_plan":"None"},"signature":"6985968131533a967fcc721244a979948b1066967f1e9cd65dbd8eeabe060fc32d894a2945f5e4a03c1cd2198c74e058ac63d28b045c2f1fcec95877bd790e1b","version":"1"}}'
```

##### step2.使用docker开启kong

```bash
docker run -d --name kong-dbless \
  --network=kong-net \
  -v "$(pwd):/kong/declarative/" \
  -e "KONG_DATABASE=off" \
  -e "KONG_DECLARATIVE_CONFIG=/kong/declarative/kong.yml" \
  -e "KONG_PROXY_ACCESS_LOG=/dev/stdout" \
  -e "KONG_ADMIN_ACCESS_LOG=/dev/stdout" \
  -e "KONG_PROXY_ERROR_LOG=/dev/stderr" \
  -e "KONG_ADMIN_ERROR_LOG=/dev/stderr" \
  -e "KONG_ADMIN_LISTEN=0.0.0.0:8001" \
  -e "KONG_ADMIN_GUI_URL=http://localhost:8002" \
  -e KONG_LICENSE_DATA \
  -p 8000:8000 \			#kong的http服务端口（kong的客户端们从这里请求kong的转发）
  -p 8443:8443 \			#kong的https服务端口
  -p 8001:8001 \			#kong的http管理端口
  -p 8444:8444 \			#kong的https的请求监控端口
  -p 8002:8002 \			#kong的管理的ui界面
  -p 8445:8445 \
  -p 8003:8003 \
  -p 8004:8004 \
  kong/kong-gateway:3.1.1.3
```

##### step3. 验证启动

1.通过kong，查看服务列表

```bash
curl -i http://localhost:8001/services
```

2.直接进入ui界面查看

> localhost:8002/overview
>
> 或者ip:8002/overview

### 1.2 使用free mode启动一个企业版本的kong

跟着这个指导即可：https://docs.konghq.com/gateway/latest/get-started/

## 2 kong的基本概念

### 2.1  service

service的作用是承接请求，并且为请求代理到生成服务时指定的url上。

有两种方式可以添加一个service：

* 向kong的admin端口（默认为8001）发送get请求，例如：

  ```bash
  curl -i -s -X POST http://localhost:8001/services \		#8001是默认管理端口
    --data name=example_service \												#希望添加的服务名称
    --data url='http://mockbin.org'											#希望服务转发到哪个url
  ```

* 通过kong的ui管理端口（默认为8002）添加。ui界面很好操作。不赘述

服务的查看，更新，删除，同样有如上两种方式，具体可见官方文档：https://docs.konghq.com/gateway/latest/get-started/services-and-routes/

### 2.2 router

router的作用是设置路由规则，kong会按照设定的规则监听端口，然后将监听到的请求进行分发。router可以绑定一个upstream，也可以绑定一个service。用于把来自用户的请求转发过去。

同样可以通过url请求方式，或者ui方式，创建router。

* 向kong的admin端口（默认为8001）发送get请求，例如：

  ```bash
  curl -i -X POST http://localhost:8001/services/example_service/routes \	#给对应的服务增加路由
    --data 'paths[]=/mock' \			#监听服务端口（8000）的/mock路径
    --data name=example_route			#router的名称
  ```

* 通过kong的ui管理端口（默认为8002）添加。ui界面很好操作。不赘述

router的crud同样可以见官方文档：https://docs.konghq.com/gateway/latest/get-started/services-and-routes/

### 2.3 service和router的关系

![服务和路线](https://docs.konghq.com/assets/images/docs/getting-started-guide/route-and-service.png)

这张图说明了router和service的关系：

router负责监听kong暴露的服务端口（默认为8000）下面的某个路径，如果有请求，则投送给该router绑定的service上。

kong中router和service的关系是N：1的，例如可以指定多个router，映射到同一个服务中。

### 2.4 upstream

> An Upstream represents a virtual hostname and can be used to load balance incoming requests over multiple Services.

即upstream表示一个虚拟host地址。这个虚拟的host地址可以用于表示若干个target。



### 2.5 target

一个target可以理解成一个需要被代理的服务endpoint。

### 2.6 upstream和target关系

![上游目标](https://docs.konghq.com/assets/images/docs/getting-started-guide/upstream-targets.png)

如图可见，一个service可以和一个实际的host绑定，同样也可以和upstream的虚拟的host绑定。然后由这个虚拟的host，连接到目标target。然后在target这里做负载均衡。

### 2.7 consumer

> A Consumer represents a User of a Service. 

可以通过指定唯一的**Username**或者唯一的**Custom ID**，来创建一个consumer。在以后的通过get api请求的过程中，如果需要对某个消费者进行特定的操作，需要在请求中加上这两个字段之一，来区分不同的用户。

创建消费者：

* 使用http创建consumer：

  ```bash
  curl -X POST http://localhost:8001/consumers/ \
    --data username=jsmith			#创建username为jsmith的消费者
  ```

* 使用ui界面创建consumer：略





## 3  kong的常规操作

### 3.1 简单url路由功能

1. 创建一个service，名称为baidu-service，设置upstream-server的host为www.baidu.com
2. 创建一个router，名称为baidu，绑定path为“/baidu”，绑定到baidu-service上
3. 使用浏览器请求localhost:8000/baidu，即可发现www.baidu.com的内容，通过kong请求后，转发到本地了

### 3.2 通过JWT插件实现身份验证和鉴权

##### 3.1.1 JWT简介

（0）用到的算法：

* HMAC-SHA256算法

  用于将长字符串进行哈希，然后转换成某个token的方式。（几乎）无法通过token解析回原字符串。可以用于检测另一个部分有没有被修改。例如A部分使用此算法生成一个token，放到A后面，下次再用A使用此算法生成token并比较前一次的即可知道有无被修改。

* Base64URL算法

  网络上最常见的用于传输8Bit字节码的编码方式之一，Base64就是一种基于64个可打印字符来表示二进制数据的方法。可以用于编码字符串为二进制。

首先了解一下JWT，看这篇文章即可：https://cloud.tencent.com/developer/article/1460770

比较重要的是以下两张图：

（1） JWT流程

![img](https://ask.qcloudimg.com/http-save/yehe-2874029/8qwls7hxns.png?imageView2/2/w/1620)

（2） JWT消息构成：

![img](https://ask.qcloudimg.com/http-save/yehe-2874029/xrvp4pjty8.png?imageView2/2/w/1620)

个人理解，JWT实际上是通过JSON格式编码鉴权，或者用于表示用户身份的信息，并且以约定的加密方式来分别加密json的以上三个部分。

* header部分相当于是控制头，用于指定编码方式，编码类型等。

* payload比较关键，里面可以存放用于区分用户身份的id等。

* signature部分，是由服务器保存的secret密钥，结合前两个字段生成的，其目的是为了保证前两个字段没有被客户端篡改。假设没有这个字段，而payload字段中又保存了用户的例如用户id信息，并且JWT Token对这部分的编码方式只是简单的Base64URL编码，并不是加密算法，所以如果没有进行二次加密，那么用户可以直接通过这个字段解析完整的payload。然后如果直接改掉用户id，再通过Base64URL编码，即可以另一个用户的身份登陆。但是有了这个字段之后，如果用户改了payload中的值，因为没有secret，所以无法生成正确的signature，所以即使发给服务器，也会被鉴定为非法请求。

  这部分的生成规则是：

  ```
   HMACSHA256(base64UrlEncode(header) + "." +  base64UrlEncode(payload),secret)
  ```

  可以看出，是结合了前面两个部分和secret，hash得出的。另一个服务收到token串后，可以进行如下操作，鉴别合法性：

  0.如果是通过https等协议，先进行https部分的解码

  1.通过“.“分割三个部分，拿到header串，payload串和signature串

  2.通过base64Url算法解码header字段，拿到其中的alg字段，这个字段表示了签名部分的生成方式。假设使用的是HMACSHA256算法。

  3.同样按照` HMACSHA256(base64UrlEncode(header) + "." +  base64UrlEncode(payload),secret)`这个规则，使用header和payload，生成signature。

  4.比较拿到的来自用户的signature串，以及2.中生成的signature串，如果相同， 说明合法。

  5.如果合法，使用base64Url算法，解码payload部分，拿到用户信息等内容。



这篇文章讲加密过程讲的很好：https://www.cnblogs.com/kirito-c/p/12402066.html

##### 3.1.2 kong使用JWT鉴权demo

* 首先先创建一个Consumer，Username为“sun_liyuan“，customID为"exiasun"

* 安装kong的jwt插件。通过ui界面或者get请求安装均可。可以给全局装，也可以只给某个服务装。

* 通过get请求，访问admin端口，通过jwt插件拿到customID为exiasun的consumer的JWT凭证：

  ```
  curl -X GET http://9.134.5.191:8001/consumers/sun_liyuan/jwt
  ```

  返回的消息是Json格式的JWT凭证：

  ```bash
  {
      "rsa_public_key": null,
      "algorithm": "HS256",
      "consumer": {
          "id": "d9235533-0d8d-439c-85dd-0a0f9456238c"
      },
      "tags": null,
      "key": "VjmTH4S8BhAhDlIptIqOWlNjbV4l6y5l",
      "id": "fd1b3dcf-efe4-489c-84e3-f00b573bac20",
      "created_at": 1675351995,
      "secret": "MJcdWgK9pfVvJd0bFSOwZokdd2Tkf1w6"
  }
  ```

  其中secret字段就是上面用于生成签名字段的secret。而key则是payload字段中的iss（谁签发了这个token）。拿到这两个字段之后，我们就可以去jwt官网上，使用这两个字段，集合其他部分， 组成上图中的JWT Token的三个部分，最终生成用于鉴权的Token：

  header字段：

  ```json
  {
    "alg": "HS256",		#指定签名部分的hash算法
    "typ": "JWT"			#鉴权方式
  }
  ```

  payload字段：

  ```json
  {
    "iss": "VjmTH4S8BhAhDlIptIqOWlNjbV4l6y5l"		#刚刚的key字段
  }
  ```

  signature字段：

  ```json
  HMACSHA256(
    base64UrlEncode(header) + "." +
    base64UrlEncode(payload),
  MJcdWgK9pfVvJd0bFSOwZokdd2Tkf1w6		#secret
  )
  ```

  使用HS256算法进行hash，最终即可生成真正的token串：

  ```
  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJWam1USDRTOEJoQWhEbElwdElxT1dsTmpiVjRsNnk1bCJ9.oB3MTVBRBOijE7BjI3jj1AcLF4o0qjK_gnAzIkWpZG0
  ```

  注意用"."分割的三部分，就是由以上三个字段hash得来的。

  * 以后通过kong代理，发请求的时候，header中增加一个`Authorization`字段即可：

    ```
    Authorization : Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJWam1USDRTOEJoQWhEbElwdElxT1dsTmpiVjRsNnk1bCJ9.oB3MTVBRBOijE7BjI3jj1AcLF4o0qjK_gnAzIkWpZG0
    ```

### 3.2 通过target实现负载均衡

按照上面的图配置：

![上游目标](https://docs.konghq.com/assets/images/docs/getting-started-guide/upstream-targets.png)

1. 创建一个upstream，名称为search，设置hostName也为search，设置负载均衡模式为轮询（round robbin）

2. 给这个upstream创建两个target，www.baidu.com和www.sogou.com

3. 创建一个service，指定名称为search-service，并且关联到名称为search，类型为host的upstream上

4. 创建一个router，关联到名称为search-service的service上，并且设置监听path为“/search”
5. 浏览器请求 localhost:8000/search，请求几次之后，会发现负载均衡到www.baidu.com和www.sogou.com上了

### 3.3 使用普罗米修斯插件实现监控

安装promethues插件即可。注意需要进行进一步的配置，可以参考：https://docs.konghq.com/hub/kong-inc/prometheus/。配置例如：

PerConsumer：是否每个consumer都监控

StatusCodeMetrics：监控http的返回码

LatencyMetrics：监控kong的延迟，上游服务延迟，请求延迟等

BandwidthMetrics：监控进出流量

UpstreamHealthMetrics：监控上游健康情况

加载插件之后，通过get8001端口的/metrcis路径，就可以拿到监控的数据。数据实例如下：

* kong_bandwidth_bytes 进出流量（label用于区分service，router，进还是出，以及consumer）

```sh
# HELP kong_bandwidth_bytes Total bandwidth (ingress/egress) throughput in bytes
# TYPE kong_bandwidth_bytes counter
kong_bandwidth_bytes{service="baidu-service",route="baidu-router",direction="egress",consumer=""} 286102
kong_bandwidth_bytes{service="baidu-service",route="baidu-router",direction="ingress",consumer=""} 3446
kong_bandwidth_bytes{service="example_service",route="example_route",direction="egress",consumer=""} 252
kong_bandwidth_bytes{service="example_service",route="example_route",direction="egress",consumer="sun_liyuan"} 7810
kong_bandwidth_bytes{service="example_service",route="example_route",direction="ingress",consumer=""} 84
kong_bandwidth_bytes{service="example_service",route="example_route",direction="ingress",consumer="sun_liyuan"} 798
kong_bandwidth_bytes{service="search-service",route="search",direction="egress",consumer=""} 1903
kong_bandwidth_bytes{service="search-service",route="search",direction="ingress",consumer=""} 1223
```

* kong_http_requests_total：http状态码统计（label用于区分service，router，http响应码，以及consumer）

  ```bash
  # HELP kong_http_requests_total HTTP status codes per consumer/service/route in Kong
  # TYPE kong_http_requests_total counter
  kong_http_requests_total{service="baidu-service",route="baidu-router",code="200",source="service",consumer=""} 4
  kong_http_requests_total{service="baidu-service",route="baidu-router",code="302",source="service",consumer=""} 2
  kong_http_requests_total{service="baidu-service",route="baidu-router",code="404",source="service",consumer=""} 1
  kong_http_requests_total{service="example_service",route="example_route",code="200",source="service",consumer="sun_liyuan"} 2
  kong_http_requests_total{service="example_service",route="example_route",code="401",source="kong",consumer=""} 1
  kong_http_requests_total{service="search-service",route="search",code="302",source="service",consumer=""} 2
  # HELP kong_kong_latency_ms Latency added by Kong and enabled plugins for each service/route in Kong
  ```

  

  

  
