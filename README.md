docker_api_test
===============

A demonstration of using Docker Engine API with golang without depending on official Docker client libraries.

This program pulls Ubuntu image from Docker Hub, creates a new container and polls it for resource usage statistics (output from `top -bn1`). Once 'q' key is pressed, run loop is cancelled, container is destroyed and program exits.

Prerequisites
-------------

* A working Docker instance. To pull the image, you need to login to Docker Hub (see `docker login --help`).
* `go-spew` (`go get -u github.com/davecgh/go-spew/spew`)
* `termbox-go` (`go get github.com/nsf/termbox-go`)

How to use
----------

Communication to Docker instance is possible in two ways:

* Through Unix socket: `./docker_api_test -unixAddr /var/run/docker.sock `
* Through a regular TCP HTTP listener: `./docker_api_test -tcpAddr 127.0.0.1:44444`. To configure your Docker instance to expose Docker Engine API through TCP, see: https://success.docker.com/article/how-do-i-enable-the-remote-api-for-dockerd. Failing that, one can leverage socat(1): 
  - `socat -d -d -d -lf ns-socat.log TCP-LISTEN:44444,reuseaddr,fork UNIX-CLIENT:/var/run/docker.sock`





