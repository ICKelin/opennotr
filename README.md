## opennotr
[![Build Status](https://travis-ci.org/ICKelin/opennotr.svg?branch=master)](https://travis-ci.org/ICKelin/opennotr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ICKelin/opennotr)](https://goreportcard.com/report/github.com/ICKelin/opennotr)

opennotr旨在提供一个简单易用的内网穿透功能，让使用者能够快速实现内网穿透，目前可以支持http，https, grpc穿透。

### opennotr是什么
opennotr是一个内网穿透项目，在[gtun](https://github.com/ICKelin/gtun)的基础之上改造而成，也是收费软件[Notr](https://www.notr.tech)的开源版本。

从技术层面，opennotr本质上是一个vpn，在底层构建的vpn基础之上，并利用了nginx的http，https和grpc的反向代理功能，另外，如果您有一定的网络技术基础，我们也支持使用coredns做动态域名解析，需要您购买自己的域名，设置ns记录指向您的coredns所在的机器的ip。

## 如何使用opennotr

**前置条件**

- 安装nginx
- 安装有iptables， iproute2等基础软件
- 一台具备公网IP的linux服务器和一台内网linux机器，配置不作太大要求。

**第一步：安装服务端程序notrd**

在公网Linux服务器上进行。

```
wget https://github.com/ICKelin/opennotr/releases/download/v0.0.1/notrd_linux_amd64
wget https://github.com/ICKelin/opennotr/releases/download/v0.0.1/notrd.toml
```

修改notrd配置文件，以下为一个实例参考
```
[server]
# notrd监听地址
listen=":9092"
# 客户端鉴权token
authKey="client server exchange key"

# vpn网络信息
[gateway]
cidr="100.64.240.1/24"
ip="100.64.240.1“

[proxy]
# nginx 配置文件目录
confDir="/etc/nginx/sites-enabled/"
# nginx证书路径
cert="/etc/nginx/certs/tls.crt"
# nginx私钥路径
key="/etc/nginx/certs/tls.key"

# 域名解析信息，可以不填
[resolver]
# etcd endpoints
etcdEndpoints=["http://localhost:2379"]%  
```

配置完成之后，启动ngnix，启动notrd
```
./notrd -conf notrd.toml
```

**第二步：运行客户端程序**

```
wget https://github.com/ICKelin/opennotr/releases/download/v0.0.1/notr_linux_amd64
wget https://github.com/ICKelin/opennotr/releases/download/v0.0.1/notr.toml
```

修改notr.toml配置文件

```
# notrd的公网地址及监听端口
serverAddr="47.115.82.137:10100"
# 步骤一当中配置的key
key="client server exchange key"
# 需要穿透的内网http端口
http=8080
# 需要穿透的内网https端口
https=4443
# 需要穿透的内网grpc端口
grpc=8800
# 需要使用的域名，如果没有域名，可配置为serverAddr。
# 如果使用域名，需要在域名供应商配置域名解析记录，解析到serverAddr
# 如果使用coredns，需要在域名供应商配置ns记录，ns记录指向coredns所在的机器的ip。
domain="47.115.82.137"
```

配置完成之后，启动notr客户端，然后启动您的内网服务，**监听的ip需要是0.0.0.0**

```
./notr -conf notr.toml
```

## 更多
如果您觉得自己搭建太麻烦，可以使用我们的[收费版本内网穿透](https://www.notr.tech)，扫描下方二维码关注之后，给公众号发送注册的用户名，待确认通过后即可免费获得30天内网穿透服务。

如果您对网络感兴趣，可以查看我的一些[文章列表](https://github.com/ICKelin/article)，或者关注我的个人公众号.

![ICKelin](qrcode.jpg)
