## opennotr
[![Build Status](https://travis-ci.org/ICKelin/opennotr.svg?branch=master)](https://travis-ci.org/ICKelin/opennotr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ICKelin/opennotr)](https://goreportcard.com/report/github.com/ICKelin/opennotr)

opennotr旨在提供一个简单易用的内网穿透功能，让使用者能够快速实现内网穿透，目前可以支持http，https穿透。如果您需要以下功能，可以考虑使用[其他版本](http://www.notr.tech)

- 快速开始，零配置，不需要购买域名和云服务器
- tcp穿透
- 节点高可用，最佳节点选择

### 依赖
- nginx
- route
- tap driver(windows)

验证nginx是否安装，验证 route/ip ro 命令是否可用

### 下载运行

**运行server**

server需要在有公网IP的服务器运行

```
wget https://github.com/ICKelin/opennotr/releases/download/v1.0.0/opennotr-server_linux

vi server.conf

{
    "device_ip": "100.64.241.1",
    "listen": ":9641",
    "tap": false,   ---------------------> 是否使用tap模式，如果客户端是windows，tap=true
    "client":[
        {
            "auth_key": "client authorize key",  ---------------> 客户端验证token
            "domain": "120.78.8.241" -----------------> 针对客户端的域名，不配置域名填server所在的服务器的公网IP
        }
    ]
}

./opennotr-server_linux -conf server.conf

验证是否运行成功
ifconfig 查看本地是否启动 tun* 网卡

```

**运行客户端**
```
./opennotr-client_darwin_amd64 -http 8000 -auth "client authorize key" -srv "opennotr-server监听的地址"

验证是否启动成功:
ping 100.64.241.1
```

## Thanks
[songgao/water](https://github.com/songgao/water)

## 最后
如果对notr感兴趣，可以关注notr的开发计划。