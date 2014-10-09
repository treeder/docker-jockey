require 'rest'

bin = "http://requestb.in/q49klpq4"

rest = Rest::Client.new()
rest.post(bin, body: "yoooooo")
