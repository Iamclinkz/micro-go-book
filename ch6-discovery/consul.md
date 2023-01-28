本章中利用consul做了服务发现中间件。discovery微服务，和string-service微服务，都通过consul注册了自己，并且暴露给consul  `serviceIP:servicePort/health` 这个url，当向这个url发送get请求，会返回当前的健康状态。例如：

```bash
# string-service的监听端口为localhost:10085
Exia➜  micro-go-book : master ✘ :✹✭ ᐅ  curl localhost:10085/health
{"status":true}
```

这样即可达到微服务的服务注册，以及服务发现的目的。具体操作如下：



#### 1.部署consul

mac可以使用brew直接安装consul，linux可以登陆官网下载对应版本（注意386为英特尔的32位cpu，amd64为英特尔和amd的64位cpu）

使用：

```bash
consul agent -dev
```

开启调试模式。这种模式下，会启动一台consul server实例，该实例自己选举自己为leader，并且只在内存中记录服务信息。不会持久化到硬盘中。shutdown之后数据不可以恢复。

部署成功后，可以在`localhost:8500`查看web界面。



#### 2.服务注册

##### 方法1：使用consul官方的包，以及go-kit包中的consul驱动，进行注册：

```golang
func NewKitDiscoverClient(consulHost string, consulPort int) (DiscoveryClient, error) {
	// 通过 Consul Host 和 Consul Port 创建一个 consul.Client
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulHost + ":" + strconv.Itoa(consulPort)
	apiClient, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
 	//把注册生成的consul client，传入go-kit中的consul驱动，让驱动帮忙管理
	client := consul.NewClient(apiClient)
	return &KitDiscoverClient{
		Host:   consulHost,
		Port:   consulPort,
		config: consulConfig,
		client: client,
	}, err
}

func (consulClient *KitDiscoverClient) Register(serviceName, instanceId, healthCheckUrl string, instanceHost string, instancePort int, meta map[string]string, logger *log.Logger) bool {

	// 1. 构建服务实例元数据
	serviceRegistration := &api.AgentServiceRegistration{
		ID:      instanceId,
		Name:    serviceName,
		Address: instanceHost,
		Port:    instancePort,
		Meta:    meta,
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30s",
			HTTP:                           "http://" + instanceHost + ":" + strconv.Itoa(instancePort) + healthCheckUrl,
			Interval:                       "15s",
		},
	}

	// 2. 发送服务注册到 Consul 中
	err := consulClient.client.Register(serviceRegistration)

	if err != nil {
		log.Println("Register Service Error!")
		return false
	}
	log.Println("Register Service Success!")
	return true
}
```

##### 方法2:直接使用consul url注册

这部分可以看`my_discover_client.go`。大概就是向`"http://"+consulClient.Host+":"+strconv.Itoa(consulClient.Port)+"/v1/agent/service/register"这个url发送注册请求。其中注册请求结构体为：

```go
// 服务实例结构体
type InstanceInfo struct {
	ID                string                     `json:"ID"`                // 服务实例ID
	Service           string                     `json:"Service,omitempty"` // 服务发现时返回的服务名
	Name              string                     `json:"Name"`              // 服务名
	Tags              []string                   `json:"Tags,omitempty"`    // 标签，可用于进行服务过滤
	Address           string                     `json:"Address"`           // 服务实例HOST
	Port              int                        `json:"Port"`              // 服务实例端口
	Meta              map[string]string          `json:"Meta,omitempty"`    // 元数据
	EnableTagOverride bool                       `json:"EnableTagOverride"` // 是否允许标签覆盖
	Check             `json:"Check,omitempty"`   // 健康检查相关配置
	Weights           `json:"Weights,omitempty"` // 权重
}
```



#### 3.向consul提供自己的健康状态

需要微服务自己实现。通过暴露  `serviceIP:servicePort/health` 这个url，当consul向这个url发送get请求，会返回当前的健康状态。



#### 4.通过consul服务发现

##### 方法1:通过go-kit包

在`kit_discover_client.go`中，不详细赘述。

##### 方法2:通过http

例如向`http://consulIP:8500/v1/health/service/string`

这个url发送Get请求，即可拿到当前的所有名称为`string`的微服务的信息。注意，返回的格式同样是上面的`InstanceInfo`类型的数组的json形式。