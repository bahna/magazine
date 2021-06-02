#!/bin/bash

echo "updating env vars"
source secret.bash

echo "serving from" $(pwd)
go run webserver/*.go -log ~/tmp/log/magazine -addr :8080 -assets ./assets -gassets ./i18n -dbhost 0.0.0.0 -debug
