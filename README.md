# Magazine CMS

Magazine CMS is a custom-built CMS developed for Bahna and deployed to 

- https://bahna.ngo
- https://bahna.land
- https://infocenter.bahna.ngo/

It uses [MongoDB](https://www.mongodb.com) as a data storage and [libvips](https://github.com/libvips/libvips) for image processing. 

`./Caddyfile` is an example of [caddy](https://caddyserver.com) configuration for the magazine server.

`./magazine.service` is an example systemctl service configuration to run the server as a service.

## Getting Started

```bash
docker compose up --build
```
