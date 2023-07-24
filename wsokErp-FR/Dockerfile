FROM golang AS builder
COPY ./erpsock /go/src/erpsock
WORKDIR /go/src/erpsock
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o foobar -ldflags="-w -s"
FROM scratch
COPY --from=builder /go/src/erpsock/foobar /erpsock
COPY --from=builder /go/src/erpsock/erp.html /erp.html
COPY --from=builder /go/src/erpsock/wsock.html /wsock.html
COPY --from=builder /go/src/erpsock/favicon.ico /favicon.ico
CMD ["/erpsock"]