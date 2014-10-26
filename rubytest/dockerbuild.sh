#!/bin/sh

docker rm rubybuild

# Need --path below because running bundler as root
docker run -i --name rubybuild -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 sh -c 'bundle install --standalone --path bundle' || docker start rubybuild

