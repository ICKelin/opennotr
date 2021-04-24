#!/bin/sh
WORKSPACE=`pwd`
BIN=$WORKSPACE/bin/
EXEC_PREFIX=opennotrd

cd $WORKSPACE/opennotrd

echo 'building client...'
GOOS=darwin go build -o $BIN/$EXEC_PREFIX-client_darwin_amd64
GOOS=linux go build -o $BIN/$EXEC_PREFIX-client_linux_amd64
GOARCH=arm GOOS=linux go build -o $BIN/$EXEC_PREFIX-client_arm
GOARCH=arm64 GOOS=linux go build -o $BIN/$EXEC_PREFIX-client_arm64

echo 'building server...'
cd $WORKSPACE/opennotrd
GOOS=linux go build -o $BIN/$EXEC_PREFIX-server_linux_amd64
