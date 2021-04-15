#!/bin/sh
WORKSPACE=`pwd`
BIN=$WORKSPACE/bin/

cd $WORKSPACE/opennotr
GOOS=darwin go build -o $BIN/opennotr-client_darwin_amd64
GOOS=linux go build -o $BIN/opennotr-client_linux_amd64
GOARCH=arm GOOS=linux go build -o $BIN/opennotr-client_arm
GOARCH=arm64 GOOS=linux go build -o $BIN/opennotr-client_arm64

cd $WORKSPACE/opennotrd
GOOS=linux go build -o $BIN/opennotr-server_linux
