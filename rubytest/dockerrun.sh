#!/bin/sh


docker rm rubyrun
docker run -i --name rubyrun -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 sh -c 'ruby yo.rb' || docker start rubyrun

