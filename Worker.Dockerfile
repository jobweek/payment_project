FROM golang:1.18-alpine as builder

RUN apk add --update --no-cache make linux-headers libc-dev gcc git gmp gmp-dev
WORKDIR /build
COPY . .
RUN make build/worker

FROM alpine:latest
WORKDIR /app/payment
COPY --from=builder /build/bin/payment/worker /app/payment/worker
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/app/payment/worker"]