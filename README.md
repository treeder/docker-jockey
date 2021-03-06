Docker Jockey - deploy what you're running, quickly.
===============

DockerJockey is a command line tool that enables you to run docker commands as you would normally, but it will fire them
up remotely on the cloud of your choice. I made this because I wanted to be able quickly take what I was working on
locally in development and get it on a box somewhere that I could point things and people at.

## Usage

1. Copy example.dj.config.json to dj.config.json and fill it in with your aws credentials.

2. Then simply take your normal docker command, for instance:

```sh
docker run -i --name mygoprog -v "$(pwd)":/app -w /app -p 8080:8080 treeder/golang-ubuntu:1.4.2on14.04 ./mygoprog
```

Where mygoprog is some program to run.

dj adds all the boilerplate for you to shorten things down:



Then change docker to `dj`:

```sh
dj deploy -i --name mygoprog -v "$(pwd)":/app -w /app -p 8080:8080 treeder/golang-ubuntu:1.4.2on14.04 ./mygoprog
```

This will fire up a machine in the cloud, get the docker image you specify, upload your code, then run it.

If you run the command again and the machine is still running, it will only upload your code and run it (much faster).

Run `dj stop CONTAINER_NAME` to stop the container and terminate the remote machine.

## Demo/Examples

You can try the gotest and rubytest examples in the appropriate directory. Basically just make sure you have a
`dj.config.json` file, then run dockerbuild.sh, then dockerrun.sh to run local, then djrun.sh to run remote.
You can use the commands inside those shell files for reference.

You can also run ready to go Docker images like:

```sh
dj run --name sinatra -p 8080:8080 treeder/sinatra
```

## Todos

The task list for this project is here: https://trello.com/b/Tm4JUSjh/jocker

## Tips

- For Ruby projects, be sure to run `bundle install --standalone` to ensure gems are packaged up.

## Building

```
go build -o dj
```

Then copy to system to test:

```
sudo cp dj /usr/local/bin/dj
```
