#!/bin/sh


docker rm gotest
docker run -i --name gotest -v "$(pwd)":/usr/src/gotest -w /usr/src/gotest -p 8080:8080 golang:1 ./gotest
