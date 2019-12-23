FROM golang:1.13-alpine

COPY . /go/src/github.com/Nivl/bl3_auto_vip
WORKDIR /go/src/github.com/Nivl/bl3_auto_vip

RUN apk add git
RUN go mod download && go mod verify

CMD go run cmd/main.go
