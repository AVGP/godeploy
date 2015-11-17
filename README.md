# godeploy
# The glue between SCM, Docker & etcd

# Building godeploy

* Go 1.5.1 is advised

```shell
$ git clone https://github.com/avgp/godeploy.git
$ cd godeploy
$ go get github.com/fsouza/go-dockerclient
$ go build -o godeploy src/main.go
```

You can now use the `godeploy` binary.

# Setting up your runtime

## Install & run etcd

This example uses `linux-amd64` as the example architecture. Pick the package that is appropriate for your server.
It also assumes `nginx` as your reverse proxy. Adapt the files accordingly to use with another reverse proxy server.

```shell
curl -L https://github.com/coreos/etcd/releases/download/v2.2.1/etcd-v2.2.1-linux-amd64.tar.gz -o etcd-v2.2.1-linux-amd64.tar.gz
tar xzvf etcd-v2.2.1-linux-amd64.tar.gz
cd etcd-v2.2.1-linux-amd64
mv etcd* /usr/bin
cd ..
rm -rf etcd*
etcd &> /var/log/etcd.log & # Replace this with a proper service file (upstart, init.d, systemd, ...)
```

## Install & run confd

```shell
wget -O /usr/bin/confd https://github.com/kelseyhightower/confd/releases/download/v0.10.0/confd-0.10.0-linux-amd64
chmod +x /usr/bin/confd
mkdir -p /etc/confd/conf.d
mkdir /etc/confd/templates
```

Create the new file `/etc/confd/conf.d/feature-branches.toml`:

```
[template]
src = "feature-branches.conf.tmpl"
dest = "/etc/nginx/sites-enabled/default.conf"
keys = [ "/deployments/" ]
reload_cmd = "/usr/sbin/service nginx reload"
```

Create the new file `/etc/confd/templates/feature-branches.conf.tmpl`:

```
server {
  listen 80 default_server;
  listen [::]:80 default_server ipv6only=on;

  root /usr/share/nginx/html;
  index index.html index.htm;
  server_name localhost;

  {{range gets "/deployments/*/*"}}
    location {{.Key}}/ {
      proxy_pass http://{{.Value}}:4567/;
    }
  {{end}}
}
```

and delete the original `/etc/nginx/sites-enabled/default`.

## Setup a Jenkins job

In Jenkins, create a new job

* Freestyle
* Git with your repository URL
* Branch specifier is `**`
* Build trigger "Poll SCM" with schedule `*/5 * * * *` to check every 5 minutes for new branches & commits
* Add build step "Shell" with the following content:

```shell
BRANCH_NAME=`echo "$GIT_BRANCH" | grep -o "[^/]*$"`
echo "Building $BRANCH_NAME..."
godeploy . YOUR_APP_NAME $BRANCH_NAME
```
(replace `YOUR_APP_NAME` with the actual name of your application)

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
