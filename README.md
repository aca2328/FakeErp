# FakeErp
Fake ERP application for demo purposes (VMware Avi Load Balancer labs).

Single Go binary serving a branded landing page (`/`), a WebSocket echo client (`/wsock.html`) and the echo endpoint (`/sock`) on port 8080.

## Build & run
```sh
./build.sh US amd64          # site: US | FR | DE, arch: amd64 | arm64
docker run -ti --rm -p 8080:8080 werp-us
```
Or run one image as any site: `docker run -e SITE=DE -p 8080:8080 werp-us`.
