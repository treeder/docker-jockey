
## Usage

TODO: copy the docker commands into this README instead of the sh scripts. 

- First run dockerbuild.sh to build the go go binary using the right docker container.
- Then run dockerrun.sh to try it locally, surf to localhost:8080 to check it out (if using boot2docker, run `boot2docker ip` to get the ip to use).
- Then run djrun.sh to run remotely. After it's running, the host name will be printed in the output. Check out
  `HOST:8080` to see it running.
