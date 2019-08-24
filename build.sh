#!/bin/sh
WORKSPACE=`pwd`
BIN=$WORKSPACE/bin/

cd $WORKSPACE/notr
go build -o $BIN/opennotr-client

cd $WORKSPACE/notrd
go build -o $BIN/opennotr-server