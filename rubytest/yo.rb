require_relative 'bundle/bundler/setup'

puts "yo"

require 'rest'

bin = "http://requestb.in/r06dwgr0"

rest = Rest::Client.new()
rest.post(bin, body: "yoooooo3")
