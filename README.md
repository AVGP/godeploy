# godeploy
# The glue between SCM, Docker & etcd

# Installation

* You'll need an etcd instance
* You'll need Docker
* Go 1.5.1 is advised

```shell
$ git clone https://github.com/avgp/godeploy.git
$ cd godeploy
$ go get github.com/fsouza/go-dockerclient
$ go build -o godeploy src/main.go
```

You can now use the `godeploy` binary.

# Usage

With etcd on the same host:

```shell
cd /your/repository/path
/path/to/godeploy . myapp somebranch
```

This will

* Remove any containers and images for `myapp:somebranch`
* Build a new image for `myapp:somebranch` from `/your/repository/path`
* Start a new container from the new image
* Announce the container IP to etcd under `/deployments/myapp/somebranch`
