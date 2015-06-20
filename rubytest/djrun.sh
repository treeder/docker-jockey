#!/bin/sh

# dj requires to escape the single quotes for now
../dj run -i --name rubyrun -v "$(pwd)":/app -w /app google/ruby sh -c \'ruby sinatra.rb\'
