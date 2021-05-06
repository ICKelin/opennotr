#!/bin/sh
WORKSPACE=`pwd`
BIN=$WORKSPACE/bin/
EXEC_PREFIX=opennotrd

cd $WORKSPACE/opennotr

echo 'building client...'
GOOS=darwin go build -o $BIN/$EXEC_PREFIX_darwin_amd64
GOOS=linux go build -o $BIN/$EXEC_PREFIX_linux_amd64
GOARCH=arm GOOS=linux go build -o $BIN/$EXEC_PREFIX_arm
GOARCH=arm64 GOOS=linux go build -o $BIN/$EXEC_PREFIX_arm64

echo 'building server...'
cd $WORKSPACE/cmd/opennotrd
GOOS=linux go build -o $BIN/$EXEC_PREFIX_linux_amd64
