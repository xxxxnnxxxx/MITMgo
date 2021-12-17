#描述
通过 `martian` 库实现的简单MITM功能

#使用说明

```shell
--contain-https --id asdfasdfasdfasd --ip 127.0.0.1 --port 8081 --hosts "[\"www.example.com\"]"

--maximumtime 180 #最大运行超时时间(分钟)
```

# 注意
在过滤https请求时，需要创建自定义的CA, 默认程序会读取当前目录下的`CA`目录， `ca.pem` 是证书， `caprikey.pem` 是私钥文件。
程序内部有专门的功能可以生成，可以自行调用