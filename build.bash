#!/bin/bash

rm -rf docker/build/src

mkdir -p docker/build/src
go mod vendor
cp -r webserver vendor go.mod go.sum docker/build/src

docker build -f docker/build/Dockerfile -t nokal/bahna-magazine-build ./docker/build
docker run --name magazine_build nokal/bahna-magazine-build
docker cp magazine_build:/gobuild/magazine-server build/
docker rm magazine_build