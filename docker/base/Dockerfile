FROM golang:stretch

# required software
RUN apt-get update && apt-get install -y vim gnupg
RUN wget -qO - https://www.mongodb.org/static/pgp/server-4.2.asc | apt-key add -
RUN echo "deb http://repo.mongodb.org/apt/debian stretch/mongodb-org/4.2 main" | tee /etc/apt/sources.list.d/mongodb-org-4.2.list
RUN apt-get update && apt-get install -y mongodb-org

WORKDIR /deploy
RUN mkdir log
RUN mkdir bin

# install vips
COPY libvips42_8.6.3-1_amd64.deb .
COPY libvips-dev_8.6.3-1_amd64.deb .
RUN apt install -y ./libvips42_8.6.3-1_amd64.deb ./libvips-dev_8.6.3-1_amd64.deb
RUN rm libvips42_8.6.3-1_amd64.deb libvips-dev_8.6.3-1_amd64.deb

CMD ["bash"]
