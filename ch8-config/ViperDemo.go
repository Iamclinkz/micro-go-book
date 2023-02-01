package main

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"reflect"
)

var Resume ResumeInformation

func init() {
	viper.AutomaticEnv()
	initDefault()
	//读取yaml文件
	//v := viper.New()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("err:%s\n", err)
	}
	//读取配置文件，并且将配置文件的内容序列化到Resume中
	if err := sub("ResumeInformation", &Resume); err != nil {
		log.Fatal("Fail to parse config", err)
	}
}
func initDefault() {
	//设置读取的配置文件
	viper.SetConfigName("resume_config")
	//添加读取的配置文件路径
	viper.AddConfigPath("./ch8-config/config/")
	//windows环境下为%GOPATH，linux环境下为$GOPATH
	viper.AddConfigPath("$GOPATH/src/")
	//设置配置文件类型
	viper.SetConfigType("yaml")
}
func main() {
	fmt.Printf(
		"姓名: %s\n爱好: %s\n性别: %s \n年龄: %d \n",
		Resume.Name,
		Resume.Habits,
		Resume.Sex,
		Resume.Age,
	)
	//反序列化
	parseYaml(viper.GetViper())
	fmt.Println(Contains("Basketball", Resume.Habits))
}

//Contains target为数组或者切片时，判断obj是否在其中
//target为map时，判断obj是否作为key
func Contains(obj interface{}, target interface{}) (bool, error) {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true, nil
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true, nil
		}
	}

	return false, errors.New("not in array")
}

type ResumeInformation struct {
	Name   string
	Sex    string
	Age    int
	Habits []interface{}
}

type ResumeSetting struct {
	RegisterTime string
	Address      string
	//ResumeInformation 和ch8-config/config/resume_config.yaml中声明的 ResumeInformation 是统一的
	ResumeInformation ResumeInformation
}

//parseYaml 从yaml中读取ResumeSetting，并且序列化到ResumeSetting结构中
func parseYaml(v *viper.Viper) {
	var resumeConfig ResumeSetting
	//方式1：直接通过传入类型，viper通过反射，填充类型中的字段
	if err := v.Unmarshal(&resumeConfig); err != nil {
		fmt.Printf("err:%s", err)
	}
	fmt.Println("resume config:\n ", resumeConfig)
}

//sub 在这里是实例的意思，读取配置文件中key
func sub(key string, value interface{}) error {
	log.Printf("配置文件的前缀为：%v", key)
	//方式2：通过传入名称，viper通过查询，直接拿到实例
	sub := viper.Sub(key)
	sub.AutomaticEnv()
	sub.SetEnvPrefix(key)
	return sub.Unmarshal(value)
}
