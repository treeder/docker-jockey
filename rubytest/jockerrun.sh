#!/bin/sh

# jocker requires to escape the single quotes for now
../jocker run -i --rm --name rubyrun -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 sh -c \'ruby yo.rb\'
