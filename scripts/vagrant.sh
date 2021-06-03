#!/bin/bash
set -e

#wget -q https://storage.googleapis.com/golang/go1.9.4.linux-amd64.tar.gz
#tar -C /usr/local -xzf go1.9.4.linux-amd64.tar.gz
#
#apt-get update
#apt-get -y install git mercurial apache2-utils

#echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin
#export GOPATH=/go' >> /home/vagrant/.profile
#
#source /home/vagrant/.profile

export GOPATH=/Users/fanhaodong/go/code/hystrix-go
export GOPROXY=https://goproxy.io,direct

#go get golang.org/x/tools/cmd/goimports
#go get github.com/golang/lint/golint
go get -u -v github.com/smartystreets/goconvey/convey
go get -u -v github.com/cactus/go-statsd-client/statsd
go get -u -v github.com/rcrowley/go-metrics
go get -u -v github.com/DataDog/datadog-go/statsd

#chown -R vagrant:vagrant /go

#echo "cd /go/src/github.com/afex/hystrix-go" >> /home/vagrant/.bashrc
