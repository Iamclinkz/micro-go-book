package conf

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	_ "github.com/streadway/amqp"
	"log"
	"net/http"
	"strings"
)

const (
	kAppName       = "APP_NAME"
	kConfigServer  = "CONFIG_SERVER"
	kConfigLabel   = "CONFIG_LABEL"
	kConfigProfile = "CONFIG_PROFILE"
	kConfigType    = "CONFIG_TYPE"
	kAmqpURI       = "AmqpURI"
)

var (
	Resume ResumeConfig
)

type ResumeConfig struct {
	Name string
	Age  int
	Sex  string
}

func init() {
	viper.AutomaticEnv()
	initDefault()
	go StartListener(viper.GetString(kAppName), viper.GetString(kAmqpURI), "springCloudBus")

	if err := loadRemoteConfig(); err != nil {
		log.Fatal("Fail to load config", err)
	}

	if err := sub("resume", &Resume); err != nil {
		log.Fatal("Fail to parse config", err)
	}
}

func initDefault() {
	//通过viper管理配置信息
	viper.SetDefault(kAppName, "client-demo")
	viper.SetDefault(kConfigServer, "http://localhost:8888")
	viper.SetDefault(kConfigLabel, "master")
	viper.SetDefault(kConfigProfile, "dev")
	viper.SetDefault(kConfigType, "yaml")
	viper.SetDefault(kAmqpURI, "amqp://admin:admin@114.67.98.210:5672")

}
func handleRefreshEvent(body []byte, consumerTag string) {
	updateToken := &UpdateToken{}
	//对来自rabbit mq的配置更新信息进行解码，如果解码正确，那么重新向spring config server拉取配置，
	//如果解码不正确，简单的忽略
	err := json.Unmarshal(body, updateToken)
	if err != nil {
		log.Printf("Problem parsing UpdateToken: %v", err.Error())
	} else {
		log.Println(consumerTag, updateToken.DestinationService)
		if strings.Contains(updateToken.DestinationService, consumerTag) {
			log.Println("Reloading Viper config from Spring Cloud Config server")
			loadRemoteConfig()
			log.Println(viper.GetString("resume.name"))
		}
	}
}

func loadRemoteConfig() (err error) {
	//把uri拼起来，然后通过get请求从配置中心中拿取对应的配置信息
	confAddr := fmt.Sprintf("%v/%v/%v-%v.%v",
		//这里是为了适配spring cloud config server的配置存储uri路径
		//spring cloud config server提供了很多层级，来管理配置信息
		//从前到后依次是：
		//config server地址/app名称/版本（对应于git的分支）-环境.文件格式
		viper.Get(kConfigServer), viper.Get(kConfigLabel),
		viper.Get(kAppName), viper.Get(kConfigProfile),
		viper.Get(kConfigType))
	resp, err := http.Get(confAddr)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	//配置viper编解码方式。刚好可以通过文件名后缀区分
	viper.SetConfigType(viper.GetString(kConfigType))
	if err = viper.ReadConfig(resp.Body); err != nil {
		return
	}
	log.Println("Load config from: ", confAddr)
	return
}

func sub(key string, value interface{}) error {
	log.Printf("配置文件的前缀为：%v", key)
	sub := viper.Sub(key)
	sub.AutomaticEnv()
	sub.SetEnvPrefix(key)
	return sub.Unmarshal(value)
}
