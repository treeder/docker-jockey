#!/bin/sh

# docker rm rubybuild
# Need --path below because running bundler as root
docker run --rm -i --name rubybuild -v "$(pwd)":/app -w /app google/ruby sh -c 'bundle install --standalone' || docker start rubybuild

