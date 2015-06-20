#!/bin/sh

docker run --rm -i -t --name rubyrun -v "$(pwd)":/app -w /app -p 8080:8080 google/ruby sh -c 'ruby sinatra.rb' || docker start rubyrun
