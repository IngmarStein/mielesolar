FROM golang:1.14.4-alpine AS builder
WORKDIR /go/src/github.com/ingmarstein/mielesolar/
COPY . .
RUN apk add -U --no-cache ca-certificates
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
COPY --from=builder /go/src/github.com/ingmarstein/mielesolar/mielesolar /mielesolar
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/mielesolar"]
