#!/bin/bash


#  if external consul node is provided use it else assume local
consul_endpoint="localhost:8500"
[ -z $1 ] || consul_endpoint=$1

# create base dir for locksmith
LOCK_BASE=/locksmith2
[ -d $LOCK_BASE ] || mkdir -p ${LOCK_BASE}
cd $LOCK_BASE

#setup go lang
[ -f ${LOCK_BASE}/go1.3.3.linux-amd64.tar.gz ] || curl -O https://storage.googleapis.com/golang/go1.3.3.linux-amd64.tar.gz
[ -d ${LOCK_BASE}/go ] || tar -zxf go1.3.3.linux-amd64.tar.gz
export PATH=$PATH:/${LOCK_BASE}/go/bin

# setup go code path
[ -d ${LOCK_BASE}/go_code/src ] ||  mkdir -p ${LOCK_BASE}/go_code/src
cd ${LOCK_BASE}/go_code/src 
export GOPATH=/${LOCK_BASE}/go_code

# pull and build dependency consul-api
[ -d ${LOCK_BASE}/go_code/src/github.com/armon/consul-api ] || go get github.com/armon/consul-api
go install github.com/armon/consul-api

# pull and build fork that implements consul
[ -d ${LOCK_BASE}/go_code/src/github.com/coreos/locksmith ] || [ -d ${LOCK_BASE}/go_code/src/github.com/npateriya/locksmith ] || go get github.com/npateriya/locksmith
[ -d ${LOCK_BASE}/go_code/src/github.com/npateriya ] && mv ${LOCK_BASE}/go_code/src/github.com/npateriya ${LOCK_BASE}/go_code/src/github.com/coreos
[ -f ${LOCK_BASE}/go_code/src/locksmithctl ] || go build github.com/coreos/locksmith/locksmithctl


# run some basic cli commands to see if its working
for cmd in status lock status unlock status 
do 
   echo;echo "==================${cmd}=============================="
   echo ${LOCK_BASE}/go_code/src/locksmithctl --backend=consul --consul-endpoint=${consul_endpoint} ${cmd}
   ${LOCK_BASE}/go_code/src/locksmithctl --backend=consul --consul-endpoint=${consul_endpoint} ${cmd}
done 
