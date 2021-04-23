#!/bin/bash
WORKSPACE=`pwd`
cd opennotrd && go build -o ../docker-build/opennotrd 

cd $WORKSPACE/docker-build
docker build . -t opennotr
