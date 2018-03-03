FROM golang:alpine

MAINTAINER Buster "Silver Eagle" Neece <buster@busterneece.com>

RUN apk update \
 && apk add git ffmpeg ca-certificates \
 && update-ca-certificates

WORKDIR /go/src/app
COPY . .

RUN CGO_ENABLED=0 go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...

CMD ["MusicBot", "-f", "bot.toml"]
