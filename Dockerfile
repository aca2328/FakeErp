FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY erpsock/go.mod erpsock/go.sum ./
RUN go mod download
COPY erpsock/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /erpsock .

FROM scratch
# Site branding: US, FR or DE (override at runtime with -e SITE=..)
ARG SITE=US
ENV SITE=${SITE}
COPY --from=builder /erpsock /erpsock
EXPOSE 8080 8443
ENTRYPOINT ["/erpsock"]
