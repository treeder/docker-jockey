#!/bin/sh

# dj requires to escape the single quotes for now
../dj run -i --rm --name rubyrun -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 sh -c \'ruby yo.rb\'
