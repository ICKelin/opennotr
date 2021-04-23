#!/bin/bash
/opt/openresty/bin/openresty -p /opt/resty-upstream -c /opt/conf/nginx-conf/nginx.conf
/opt/opennotrd -conf /opt/conf/notrd.yaml
