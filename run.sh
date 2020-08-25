#!/bin/sh

TagVersion=v1.15.0
IMAGE=envoyproxy/envoy-alpine:${TagVersion}
NAME=envoy-${TagVersion}

cd "$(dirname "$0")" || exit 1

if  [ $# -ne 1 ];then
	echo "must choose one config file"
	exit 1
fi

config=$1

docker rm -f $NAME 2>&1 1>/dev/null

if [ `uname` = "Darwin" ];then
	docker run -idt --name $NAME -p 9901:9901 -p 1080:1080 -v `pwd`/$config:/etc/envoy/envoy.yaml -v `pwd`/log:/var/log/envoy $IMAGE
else
	docker run -idt --name $NAME -e loglevel=debug --network=host -v `pwd`/$config:/etc/envoy/envoy.yaml -v `pwd`/log:/var/log/envoy $IMAGE
fi
