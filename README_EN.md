## opennotr
[![Build Status](https://travis-ci.org/ICKelin/opennotr.svg?branch=master)](https://travis-ci.org/ICKelin/opennotr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ICKelin/opennotr)](https://goreportcard.com/report/github.com/ICKelin/opennotr)

opennotr is a nat tranversal application base on a VPN tunnel and openresty.

opennotr provides http, https, grpc, tcp and udp nat traversal. For http, https, grpc, opennotr supports multi client share the 80/443 ports, it maybe useful for wechat, facebook webhook debug.

The technical architecture of opennotr

![opennotr.jpg](opennotr.jpg)

Table of Contents
=================
- [Features](#Features)
- [Build](#build)
- [Install](#Install)
- [Technology details](#Technology-details)
- [Author](#Author)

Features
=========
opennotr provides these features:

- Supports multi protocol, http, https, grpc, tcp, udp.
- Multi client shares the same http, https, grpc port, for example: client A use `a.notr.tech` domain, client B use `b.notr.tech`, they can both use 80 port for http. Opennotr use openresty for dynamic upstream.
- Dynamic dns support, opennotr use coredns and etcd for dynamic dns.

[Back to TOC](#table-of-contents)

Build
=====

**Build binary:**

`./build_exec.sh`

The binary file will created in bin folder.

**Build docker image:**

`./build_image.sh`

This scripts will run `build_exec.sh` and build an image name `opennotr`

[Back to TOC](#table-of-contents)

Install
=========

**Install via docker-compose**

1. create configuration file

`mkdir /opt/data/opennotrd`

An example of configuration folder tree is:

```
root@iZwz97kfjnf78copv1ae65Z:/opt/data/opennotrd# tree
.
|-- cert ---------------------> cert folder
|   |-- upstream.crt
|   `-- upstream.key
`-- notrd.yaml ---------------> opennotr config file

2 directories, 5 files
```

the cert folder MUST be created and the crt and key file MUST created too.

```
root@iZwz97kfjnf78copv1ae65Z:/opt/data/opennotrd# cat notrd.yaml
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

the only one configuration item you should change is `domain: "open.notr.tech"`, replace `open.notr.tech` with your own domain.

2. Run with docker

`docker run --privileged --net=host -v /opt/logs/opennotr:/opt/resty-upstream/logs -v /opt/data/opennotrd:/opt/conf -d opennotr`

Or use docker-compose


```
wget https://github.com/ICKelin/opennotr/blob/develop/docker-build/docker-compose.yaml

docker-compose up -d opennotrd
```

[Back to TOC](#table-of-contents)

Technology details
==================

- [opennotr architecture]()
- [opennotr dynamic upstream implement]()
- [opennotr vpn implement]()

[Back to TOC](#table-of-contents)

Author
======
A programer name ICKelin.