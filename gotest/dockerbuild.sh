#!/bin/sh

docker rm gobuild

docker run -i --name gobuild -v "$(pwd)":/usr/src/gotest -w /usr/src/gotest golang:1 go build || docker start gobuild

#docker run --rm  -v "$(pwd)":/usr/src/gotest -w /usr/src/gotest golang:1 go build
