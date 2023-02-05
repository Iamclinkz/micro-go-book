package service

import (
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/hashicorp/consul/api"
	"github.com/longjoy/micro-go-book/ch10-resiliency/use-string-service/config"
	"github.com/longjoy/micro-go-book/common/discover"
	"github.com/longjoy/micro-go-book/common/loadbalance"
	"net/http"
	"net/url"
	"strconv"
)

// Service constants
const (
	StringServiceCommandName = "String.string"
	StringService            = "string"
)

var (
	ErrHystrixFallbackExecute = errors.New("hystrix fall back execute")
)

// Service Define a service interface
type Service interface {
	// 远程调用 string-service 服务
	UseStringService(operationType, a, b string) (string, error)

	// 健康检查
	HealthCheck() bool
}

//ArithmeticService implement Service interface
type UseStringService struct {
	// 服务发现客户端
	discoveryClient discover.DiscoveryClient
	loadbalance     loadbalance.LoadBalance
}

func NewUseStringService(client discover.DiscoveryClient, lb loadbalance.LoadBalance) Service {

	hystrix.ConfigureCommand(StringServiceCommandName, hystrix.CommandConfig{
		// 设置触发最低请求阀值为 5，方便我们观察结果
		RequestVolumeThreshold: 5,
	})
	return &UseStringService{
		discoveryClient: client,
		loadbalance:     lb,
	}

}

// StringResponse define response struct
type StringResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

//这里在service层，自己的service内部封装了对hystrix的使用，也可以使用go-kit包，在endpoint层使用hystrix
//func (s UseStringService) UseStringService(operationType, a, b string) (string, error) {
//
//	var operationResult string
//
//	//这里调用hystrix.Do，同步使用hystrix代理执行我们的请求，需要传入一个string，表示本次命令的名称，
//	//相同命令的调用，会使用相同的断路器。这里如果hystrix该命令字的断路器已经打开，那么不会执行远程调用过程
//	//直接返回错误
//	err := hystrix.Do(StringServiceCommandName, func() error {
//		instances := s.discoveryClient.DiscoverServices(StringService, config.Logger)
//		// 随机选取一个服务实例进行计算
//		instanceList := make([]*api.AgentService, len(instances))
//		for i := 0; i < len(instances); i++ {
//			instanceList[i] = instances[i].(*api.AgentService)
//		}
//		// 使用负载均衡算法选取实例
//		selectInstance, err := s.loadbalance.SelectService(instanceList)
//		if err != nil {
//			config.Logger.Println(err.Error())
//			return err
//		}
//		config.Logger.Printf("current string-service ID is %s and address:port is %s:%s\n", selectInstance.ID, selectInstance.Address, strconv.Itoa(selectInstance.Port))
//		requestUrl := url.URL{
//			Scheme: "http",
//			Host:   selectInstance.Address + ":" + strconv.Itoa(selectInstance.Port),
//			Path:   "/op/" + operationType + "/" + a + "/" + b,
//		}
//
//		resp, err := http.Post(requestUrl.String(), "", nil)
//		if err != nil {
//			return err
//		}
//		result := &StringResponse{}
//
//		err = json.NewDecoder(resp.Body).Decode(result)
//		if err != nil {
//			return err
//		} else if result.Error != nil {
//			return result.Error
//		}
//
//		operationResult = result.Result
//		return nil
//
//	}, func(e error) error {
//		//这里可以定义服务错误（包括断路器打开）之后的处理方式（例如回滚操作）
//		return ErrHystrixFallbackExecute
//	})
//	return operationResult, err
//}

//UseStringService 普通版本的对string-service的调用，没有容错处理。
//如果string-service服务有问题，且同一时刻有很多客户端调用本服务，从而给string-service发送很多个
//Post请求，那么同时有很多个go程挂起，从而影响到本服务。
//具体流程如下：
//1.通过consul拿到所有提供string-service的服务实例（api.AgentService）类型
//2.通过负载均衡算法，从中选取一个实例
//3.发送http请求，调用string-service
func (s UseStringService) UseStringService(operationType, a, b string) (string, error) {

	var operationResult string
	var err error

	instances := s.discoveryClient.DiscoverServices(StringService, config.Logger)
	instanceList := make([]*api.AgentService, len(instances))
	for i := 0; i < len(instances); i++ {
		instanceList[i] = instances[i].(*api.AgentService)
	}
	// 使用负载均衡算法选取实例
	selectInstance, err := s.loadbalance.SelectService(instanceList)
	if err == nil {
		config.Logger.Printf("current string-service ID is %s and address:port is %s:%s\n", selectInstance.ID, selectInstance.Address, strconv.Itoa(selectInstance.Port))
		requestUrl := url.URL{
			Scheme: "http",
			Host:   selectInstance.Address + ":" + strconv.Itoa(selectInstance.Port),
			Path:   "/op/" + operationType + "/" + a + "/" + b,
		}

		resp, err := http.Post(requestUrl.String(), "", nil)
		if err == nil {
			result := &StringResponse{}
			err = json.NewDecoder(resp.Body).Decode(result)
			if err == nil && result.Error == nil {
				operationResult = result.Result
			}

		}
	}
	return operationResult, err
}

// HealthCheck implement Service method
// 用于检查服务的健康状态，这里仅仅返回true。
func (s UseStringService) HealthCheck() bool {
	return true
}

// ServiceMiddleware define service middleware
type ServiceMiddleware func(Service) Service
