FROM nokal/bahna:latest

# sensitive variables
ARG hash
ARG block
ARG secret
ENV BAHNA_HASH_KEY=$hash
ENV BAHNA_BLOCK_KEY=$block
ENV BAHNA_SECRET=$secret

# magazine build
RUN mkdir -p /deploy/bin
WORKDIR /devel
RUN mkdir cmd
COPY cmd/magazine cmd/magazine
COPY appkit .
COPY cms .
COPY go.mod .
COPY go.sum .
COPY mail .
COPY vendor .
RUN go build -mod vendor -o /deploy/bin/magazine-server cmd/magazine

WORKDIR /deploy
RUN mkdir log

# global assets
COPY assets ./cms_assets

# magazine assets
RUN mkdir -p magazine/assets
COPY cmd/magazine/assets/files magazine/assets
COPY cmd/magazine/assets/scripts magazine/assets
COPY cmd/magazine/assets/static magazine/assets
COPY cmd/magazine/assets/styles magazine/assets
COPY cmd/magazine/assets/templates magazine/assets

# magazine database
RUN mongorestore -d magazine /devel/db-dumps/magazine

# magazine service
COPY magazine.service /etc/systemd/system
RUN systemctl daemon-reload
RUN systemctl restart magazine

# caddy
COPY caddy_linux_amd64_custom/caddy bin/
COPY Caddyfile .

CMD ["bin/caddy", "-f", "Caddyfile"]
