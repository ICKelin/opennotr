## opennotr
[![Build Status](https://travis-ci.org/ICKelin/opennotr.svg?branch=master)](https://travis-ci.org/ICKelin/opennotr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ICKelin/opennotr)](https://goreportcard.com/report/github.com/ICKelin/opennotr)

opennotr旨在提供一个简单易用的内网穿透功能，让使用者能够快速实现内网穿透，目前可以支持http，https, grpc穿透。

opennotr本质上是一个vpn，在底层构建的vpn基础之上，集成了coredns作为域名解析模块，并利用了nginx的http，https和grpc的反向代理功能。

## 安装运行

### 运行server端
server端需要运行在具备公网ip的服务器上，server作为各个客户端的**虚拟路由器**，负责vhost的分配，代理，客户端长连接隧道等。

**安装依赖**

- nginx
- iproute2
- coredns+etcd

确保服务器能够正确执行`ifconfig`,`ip ro`命令

配置文件实例:

config.toml
```
[server]
listen="127.0.0.1:9092" # server监听地址
authKey="client server exchange key" #校验客户端的token
domain="open.notr.tech" # 系统使用的域名

[gateway]
cidr="192.168.100.1/24" # ip地址分配
ip="192.168.100.1" # 网关地址/本地地址

[proxy]
confDir="/etc/nginx/sites-enabled/" # nginx配置目录
cert="/etc/nginx/certs/tls.crt" # https证书路径
key="/etc/nginx/certs/tls.key" # https秘钥路径

[resolver]
etcdEndpoints=["http://localhost:2379"]  # etcd配置

```

```
sudo ./opennotrd -conf config.toml
```

### 运行client
client只支持linux，确保服务器能够正确执行`ifconfig`,`ip ro`命令

配置文件实例
config.toml
```
serverAddr="holenat.net:9092" # server地址
key="client server exchange key" # client验证key
http=8080 # 本地http端口
https=4443 # 本地https端口
grpc=8800 # 本地grpc端口

```

```
sudo ./opennotr -conf config.toml

```


运行成功之后，客户端对应的http，https，grpc服务监听的ip是`0.0.0.0`，不能是`127.0.0.1`

## opennotr工作原理

opennotr构建在一个vpn基础上，所有client都和server组成了一个虚拟局域网，每个客户端会被分配一个局域网ip，在server端通过客户端的虚拟局域网ip访问客户端。

基于此基础，在server端利用nginx的反向代理能力，代理地址为客户端的虚拟ip地址，底层的vpn功能针对nginx而言是透明的，对nginx不可见。

为了适配ip地址的变更，同时也为了能够充分利用80和443端口，引入了coredns作为域名解析服务器，不仅为每个客户端分配一个虚拟局域网ip，也将虚拟局域网ip解析为域名，因此，无论底层客户端的虚拟局域网ip如何发生变化，使用域名访问同样是透明的，使用者并不需要关注底层虚拟局域网的概念。

## Thanks
[songgao/water](https://github.com/songgao/water)

如果您觉得这个项目不错，可以给star，如果有更好的意见和建议，或者希望对源码进行改造，可以与我取得联系。