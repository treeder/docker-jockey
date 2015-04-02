Jocker - The Docker Jockey
===============

Jocker is a command line tool that enables you to run docker commands as you would normally, but it will fire them
up remotely on the cloud of your choice. I made this because I wanted to be able quickly take what I was working on
locally in development and get it on a box somewhere that I could point things and people at.

## Usage

1. Copy example.jocker.config.json to jocker.config.json and fill it in with your aws credentials.

2. Then simply take your normal docker command, for instance:

```sh
dj run -i --name mygoprog -v "$(pwd)":/app -w /app -p 8080:8080 treeder/golang-ubuntu:1.3.3on14.04 ./mygoprog
```

Where mygoprog is some program to run. 

Then change docker to `dj`:

```sh
dj run -i --name mygoprog -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp -p 8080:8080 treeder/golang-ubuntu:1.3.3on14.04 ./mygoprog
```

This will fire up a machine in the cloud, get the docker image you specify, upload your code, then run it.

If you run the command again and the machine is still running, it will only upload your code and run it (much faster).

Run `jocker stop CONTAINER_NAME` to stop the container and terminate the remote machine.

## Demo/Examples

You can try the gotest and rubytest examples in the appropriate directory. Basically just make sure you have a jocker.config.json
file then run dockerbuild.sh, then dockerrun.sh to run local, then jockerrun.sh to run remote. You can use the commands
inside those shell files for reference.

You can also run ready to go Docker images like:

```sh
dj run --name ironmq -p 8080:8080 iron/mq
```

## Todos

The task list for this project is here: https://trello.com/b/Tm4JUSjh/jocker

## Tips

- For Ruby projects, be sure to run `bundle install --deployment` to ensure gems are packaged up.

