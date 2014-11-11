#!/bin/sh

../jocker run -i --rm --name gotest -v "$(pwd)":/usr/src/gotest -w /usr/src/gotest -p 8080:8080 golang:1 ./gotest
