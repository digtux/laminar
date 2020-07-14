FROM golang:1.14 as BUILD

WORKDIR /build
COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o laminar .


FROM alpine

RUN apk add --no-cache git bash
WORKDIR /
COPY --from=BUILD /build/laminar .
CMD "/laminar"
