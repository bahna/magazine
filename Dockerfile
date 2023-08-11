FROM --platform=linux/arm64 golang:bullseye AS build

# vips has to be present for the build
RUN apt-get update && apt-get install -y libvips-dev

RUN mkdir /app
WORKDIR /app
ADD . .
RUN git config --global --add safe.directory /app
RUN go build -mod=vendor -v -o magazine-server github.com/bahna/magazine/webserver

FROM debian:bullseye-slim AS deploy

RUN apt-get update && apt-get install -y libvips-dev

RUN mkdir /deploy
COPY --from=build /app/magazine-server /deploy/magazine-server

WORKDIR /deploy
ADD ./assets ./assets
ADD ./i18n ./i18n
ADD ./secret.bash ./secret.bash

EXPOSE 8080
