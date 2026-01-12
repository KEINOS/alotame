FROM golang:alpine AS builder

RUN \
    apk update --no-cache && \
    apk upgrade --no-cache

FROM builder

WORKDIR /app

COPY . .

RUN go build -o alotame main.go

EXPOSE 5963

ENTRYPOINT [ "/app/alotame" ]
