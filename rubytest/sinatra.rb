require_relative 'bundle/bundler/setup'
require 'sinatra'

set :port, 8080
set :bind, '0.0.0.0'

post('/somepost') do
  puts "in somepost"
  p params
end
get('/ping') { "pong" }
get('/') { "hi" }

