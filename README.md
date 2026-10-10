# FakeErp
Fake ERP application for demo purposes (VMware Avi Load Balancer labs).

Single Go binary serving a branded landing page (`/`), a WebSocket echo client (`/wsock.html`) and the echo endpoint (`/sock`) on port 8080.

## Build & run
```sh
./build.sh US amd64          # site: US | FR | DE, arch: amd64 | arm64
docker run -ti --rm -p 8080:8080 werp-us
```
Or run one image as any site: `docker run -e SITE=DE -p 8080:8080 werp-us`.

## Lab endpoints
| Path | Use |
|---|---|
| `/info` | JSON: backend hostname/IP, site, client IP, `X-Forwarded-*`, all headers — shows load balancing and client IP insertion |
| `/health`, `/health/fail`, `/health/ok` | Health monitor target; flip a backend down/up to demo failover |
| `/slow?ms=2000` | Delayed response (max 60 s) — latency analytics, fastest-response LB |
| `/error?code=503` | Return any 4xx/5xx — error analytics, alerts |
| `/cpu?ms=500` | Burn CPU (max 10 s) — server load / autoscale |
| `/search?q=`, `/login` | Fake SQL/XSS-looking forms for WAF demos (input is escaped, no real DB) |
| `/wsock.html`, `/sock` | WebSocket echo client / endpoint |

## Kubernetes (AKO)
`kubectl apply -f k8s/werp.yaml` deploys US/FR/DE sites (3 replicas each) behind an `avi-lb` Ingress on `erp-{us,fr,de}.lab.local`. Edit the image name and hosts for your lab.
