Jocker - The Docker Jockey
===============

Jocker is a command line tool that enables you to run docker commands as you would normally, but it will fire them
up remotely on the cloud of your choice. I made this because I wanted to be able quickly take what I was working on
locally in development and get it on a box somewhere that I could point things and people at.

# Usage

1. Copy example.jocker.config.json to jocker.config.json and fill it in with your aws credentials.

2. Then simply take your normal docker command, for instance:

```sh
docker run -i --name mygoprog -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp -p 8080:8080 treeder/golang-ubuntu:1.3.3on14.04 ./mygoprog
```

And change it to `jocker`:

```sh
jocker run -i --name mygoprog -v "$(pwd)":/usr/src/myapp -w /usr/src/myapp -p 8080:8080 treeder/golang-ubuntu:1.3.3on14.04 ./mygoprog
```

This will fire up a machine in the cloud, get the docker image you specify, upload your code, then run it.

If you run the command again and the machine is still running, it will only upload your code and run it (much faster).

## Todos

The task list for this project is here: https://trello.com/b/Tm4JUSjh/jocker

## Tips

- For Ruby projects, be sure to run `bundle install --deployment` to ensure gems are packaged up.

