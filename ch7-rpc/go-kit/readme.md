正如上一章的readme.md中所说，endpoint这一层定义了输入和输出的格式。输入输出往往不需要同编码强关联。例如`ch6-discovery/string-service/endpoint/endpoints.go`中结构体，例如：

```go
// StringRequest define request struct
type StringRequest struct {
	//注意声明的时候可以加上tag，用来自定义json的解码
	RequestType string `json:"request_type"`
	A           string `json:"a"`
	B           string `json:"b"`
}
```

​	虽然通过tag，定义了json转换格式，但是实际上仍然是结构体。根json本身没有关系。

本章也一样。`ch7-rpc/go-kit/string-service/endpoints.go`中，同样定义了输入和输出的格式，只不过是通过protoc工具自动生成的go结构，例如：

```go
type StringResponse struct {
	Ret                  string   `protobuf:"bytes,1,opt,name=Ret,proto3" json:"Ret,omitempty"`
	Err                  string   `protobuf:"bytes,2,opt,name=err,proto3" json:"err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}
```

上层（在这两个例子中，都是transport层）均需要负责编解码。千万不要看结构体的tag，武断的觉得上层使用什么编解码方式。例如上面两个结构体，第一个也可以用protoc编码，第二个也可以用json编码。具体的编解码，以及通信协议，应该交给transport这一层。

endpoint这一层负责定义输入输出，同时wrapper设计模式，用洋葱模型，包裹住Service层。