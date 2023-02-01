#### 在目录ch9-gateway下面执行本脚本

1. 启动consul
```bash
consul agent -dev
```

2. 启动反向代理
```bash
cd gateway
go run main.go
```

3. 启动string-service，它会自动注册到consul中
```bash
cd string-service
go build
./string-service
```



