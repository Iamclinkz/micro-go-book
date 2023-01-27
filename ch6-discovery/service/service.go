package service

import (
	"context"
	"errors"
	"github.com/longjoy/micro-go-book/ch6-discovery/config"
	"github.com/longjoy/micro-go-book/ch6-discovery/discover"
)

//Service 借助DiscoveryClient的能力，实现服务发现服务
type Service interface {

	// HealthCheck check service health status
	HealthCheck() bool

	// sayHelloService
	SayHello() string

	//  discovery service from consul by serviceName
	DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error)
}

var ErrNotServiceInstances = errors.New("instances are not existed")

//DiscoveryServiceImpl 利用内部的DiscoveryClient，实现了Service接口，即对外提供服务发现接口
type DiscoveryServiceImpl struct {
	//内部持有一个服务发现客户端实例，对外提供服务发现服务。
	discoveryClient discover.DiscoveryClient
}

func NewDiscoveryServiceImpl(discoveryClient discover.DiscoveryClient) Service {
	return &DiscoveryServiceImpl{
		discoveryClient: discoveryClient,
	}
}

func (*DiscoveryServiceImpl) SayHello() string {
	return "Hello World!"
}

//DiscoveryService 传入serviceName，返回提供服务的实例
func (service *DiscoveryServiceImpl) DiscoveryService(ctx context.Context, serviceName string) ([]interface{}, error) {

	//直接使用了内部持有的discoveryClient的DiscoverServices方法
	instances := service.discoveryClient.DiscoverServices(serviceName, config.Logger)

	if instances == nil || len(instances) == 0 {
		return nil, ErrNotServiceInstances
	}
	return instances, nil
}

// HealthCheck implement Service method
// 用于检查服务的健康状态，这里仅仅返回true
func (*DiscoveryServiceImpl) HealthCheck() bool {
	return true
}
