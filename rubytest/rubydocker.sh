#!/bin/sh

#docker run -it --rm --name my-running-script -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 bundle exec ruby yo.rb
../jocker run -it --rm --name my-running-script -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp ruby:2 bundle exec ruby yo.rb
