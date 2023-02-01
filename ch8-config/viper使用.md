###  拿到解析好的配置

首先要明确，配置映射到go中，可能是以下形式：

* key-value对形式（例如 env:dev）
* 结构体形式（例如Person下面有若干字段，每个字段有一个值）

所以我们可以直接通过类型名称取出，也可以通过key取出。



##### 方式1：直接通过传入类型，viper通过反射，填充类型中的字段

例如yaml中的配置文件为：

```yaml
RegisterTime: "2019-6-18 10:00:00"
Address: "Shanghai"
ResumeInformation:
  Name: "aoho"
  Sex: "male"
  Age: 20
  Habits:
    - "Basketball"
    - "Running"
```

虽然最顶层的结构没有名称，但是其他字段都有名称。而如果我们传入一个go的与之对应的结构体：

```go
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
```

go实际上是可以通过拿到各个字段的名称，来判断go结构体是否和yaml中的结构体一致的。

所以可以直接传入一个go结构体，让viper填充，例如：

```go
//parseYaml 从yaml中读取ResumeSetting，并且序列化到ResumeSetting结构中
func parseYaml(v *viper.Viper) {
	var resumeConfig ResumeSetting
	//方式1：直接通过传入类型，viper通过反射，填充类型中的字段
	if err := v.Unmarshal(&resumeConfig); err != nil {
		fmt.Printf("err:%s", err)
	}
	fmt.Println("resume config:\n ", resumeConfig)
}
```

虽然go中的结构体声明了之后，可以实例化很多个实例，但是配置中的key-value对应该是唯一的。所以如果用一个结构体来表示一个配置，那么这个结构体表示的配置也是唯一确定的。



##### 方式2：通过某个字段的名称

还是上面的yaml文件，实际上yaml文件中的每个字段，映射到go中都是一个键值对（key一定为string，但是value可以为string，也可以为key-value对的集合）

所以我们可以通过传入key来查询配置。例如：

```go
//sub 在这里是实例的意思，读取配置文件中key
func sub(key string, value interface{}) error {
	log.Printf("配置文件的前缀为：%v", key)
	//方式2：通过传入名称，viper通过查询，直接拿到实例
	sub := viper.Sub(key)
	sub.AutomaticEnv()
	sub.SetEnvPrefix(key)
  //实例反序列化，填充ResumeInformation类型的实例
	return sub.Unmarshal(value)
}
```

