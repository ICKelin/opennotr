#!/bin/sh
WORKSPACE=`pwd`
BIN=$WORKSPACE/bin/

cd $WORKSPACE/notr
GOOS=darwin go build -o $BIN/opennotr-client_darwin_amd64
GOOS=linux go build -o $BIN/opennotr-client_linux_amd64
GOARCH=arm GOOS=linux go build -o $BIN/opennotr-client_arm.exe
GOARCH=arm64 GOOS=linux go build -o $BIN/opennotr-client_arm64.exe

cd $WORKSPACE/notrd
GOOS=linux go build -o $BIN/opennotr-server_linux
