#!/bin/bash

name="magazine"

echo "updating env vars"
source secret.sh
echo "serving from" $(pwd)
go run cmd/${name}/*.go -log ~/tmp/log/${name} -addr :8080 -assets cmd/${name}/assets -gassets assets -dbhost 0.0.0.0 -debug
