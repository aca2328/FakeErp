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
HTTP on `:8080`, HTTPS + HTTP/2 on `:8443` (self-signed cert, or set `TLS_CERT`/`TLS_KEY`).

| Path | Use |
|---|---|
| `/info` | JSON: backend hostname/IP, site, client IP, `X-Forwarded-*`, all headers — load balancing, client IP insertion |
| `/health`, `/health/fail`, `/health/ok` | Health monitor target; flip a backend down/up to demo failover |
| `/session` | Cookie `ERPSESSION`, stored per backend — shows "SESSION LOST" when persistence is missing |
| `/sales`, `/stock`, `/accounting` | Mock ERP pages — content switching, rewrites, per-path policies |
| `/api/{sales,stock,accounting}?limit=N` | JSON API with fake data (max 1000 rows) |
| `/slow?ms=2000` | Delayed response (max 60 s) — latency analytics, fastest-response LB |
| `/error?code=503` | Return any 4xx/5xx — error analytics, alerts |
| `/cpu?ms=500` | Burn CPU (max 10 s) — server load / autoscale |
| `/search?q=`, `/login` | Fake SQL/XSS-looking forms for WAF demos (input escaped, no real DB). `demo`/`demo` logs in |
| `/wsock.html`, `/sock` | WebSocket client (backend, RTT, counters, auto-reconnect) / echo endpoint (`?hello=1` announces backend) |
| `/metrics` | Prometheus metrics: requests per route/code, latency, WebSocket connections, health |

Access logs (client, `X-Forwarded-For`, request, status, size, duration) go to stdout; disable with `ACCESS_LOG=0`.

## Load generator
Built into the same binary/image:
```sh
docker run --rm werp-us loadgen -c 20 -d 5m http://vip.lab.local     # -k: skip TLS verify, -attacks: add SQLi/XSS for WAF
```

## Kubernetes (AKO)
`kubectl apply -f k8s/werp.yaml` deploys US/FR/DE sites (3 replicas each, HTTP 80 / HTTPS 443 services, Prometheus annotations) behind an `avi-lb` Ingress on `erp-{us,fr,de}.lab.local`. `k8s/loadgen.yaml` runs the load generator as a Job. Edit image names and hosts for your lab.
