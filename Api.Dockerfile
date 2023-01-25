FROM golang:1.18-alpine as builder

RUN apk add --update --no-cache make linux-headers libc-dev gcc git gmp gmp-dev
WORKDIR /build
COPY . .
RUN make build/api

FROM alpine:latest
WORKDIR /app/payment
COPY --from=builder /build/bin/payment/api /app/payment/main
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8088

ENTRYPOINT ["/app/payment/main"]