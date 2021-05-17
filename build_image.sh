#!/bin/bash
./build_exec.sh

cd $WORKSPACE/docker-build
docker build . -t opennotrd
