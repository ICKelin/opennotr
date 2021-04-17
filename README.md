## opennotr
[![Build Status](https://travis-ci.org/ICKelin/opennotr.svg?branch=master)](https://travis-ci.org/ICKelin/opennotr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ICKelin/opennotr)](https://goreportcard.com/report/github.com/ICKelin/opennotr)

opennotr is a nat tranversal application base on layer 3 tunnel and openresty.

opennotr construct a layer3 VPN(we use tun device which is widly used by OpenVPN) and offer a LAN IPV4 address for each client.

For gateway, we use openresty and set upstream via http API. And proxy the http, https. grcp(tcp/udp is on the way) traffic to the client LAN IPV4 address.


## Install via docker
configuration directory
```
/opt/data/opennotrd/
|-- nginx-conf ----------> openresty configuration
|   |-- cert ------------> https cert
|   |   |-- upstream.crt
|   |   `-- upstream.key
|   |-- mime.types
|   `-- nginx.conf 
`-- notrd.yaml -----------> opennotrd application configuration
```

**configuration example**

**notrd.yaml**

```
# cat /opt/data/opennotrd/notrd.yaml
server:
  listen: ":10100"
  authKey: "client server exchange key"
  domain: "open.notr.tech"

dhcp:
  cidr: "100.64.242.1/24"
  ip: "100.64.242.1"

upstream:
  remoteAddr: "http://127.0.0.1:81/upstreams"
```
upstream.remoteAddr is the dynami upstram server base on [openresty](https://openresty.org)

**nginx-config** 

nginx-config is openresty conf folder, here is an example [nginx.conf](https://github.com/ICKelin/resty-upstream/blob/master/conf/nginx.conf)

**install via docker**
```
docker run --privileged --net=host -v /opt/logs/opennotr:/opt/resty-upstream/logs -v /opt/data/opennotrd:/opt/conf -d ickelin/opennotr:latest
```
