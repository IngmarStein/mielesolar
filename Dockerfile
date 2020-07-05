FROM golang:1.14.4 AS builder
WORKDIR /go/src/github.com/ingmarstein/mielesolar/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
ENV INVERTER_ADDRESS=192.168.188.167
ENV INVERTER_PORT=502

COPY --from=builder /go/src/github.com/ingmarstein/mielesolar/mielesolar .
ENTRYPOINT ["./mielesolar"]
