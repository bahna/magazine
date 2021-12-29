# Magazine CMS

Magazine CMS is a custom-built CMS developed for Bahna and deployed to 

- https://bahna.ngo
- https://bahna.land
- https://infocenter.bahna.ngo/

It uses [MongoDB](https://www.mongodb.com) as a data storage and [libvips](https://github.com/libvips/libvips) for image processing. Because of the _libvips_ dependency, the software can be built **only** under Linux. For this reason, a  docker container is provided.

To build, execute the `./build.bash` script. It requires the pre-built `nokal/magazine_base` docker image, which should be available from [hub.docker.com](https://hub.docker.com/r/nokal/magazine_base) or one can build it from `./docker/base`. 

`./Caddyfile` is an example of [caddy](https://caddyserver.com) configuration for the magazine server.

`./magazine.service` is an example systemctl service configuration to run the server as a service.
