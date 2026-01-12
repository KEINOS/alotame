# Usage:
#   $ docker build -t test .
#   $ docker run test 1.1.1.1:53 yahoo.com dns.example.com
#   [
#       {
#         "domain": "yahoo.com",
#         "status": "ALLOWED",
#         "detail": "98.137.11.164"
#       },
#       {
#         "domain": "dns.example.com",
#         "status": "BLOCKED",
#         "detail": "no answer"
#       }
#   ]


FROM golang:alpine AS builder

RUN \
    apk update --no-cache && \
    apk upgrade --no-cache

FROM builder

WORKDIR /app

COPY . .

RUN \
    go get -u ./... && \
    go mod tidy

RUN go build -o "/app/dns-resolver-checker" ./...

ENTRYPOINT [ "/app/dns-resolver-checker"]
