FROM golang:alpine AS builder

ADD . /go/drone-deb-simple
RUN cd drone-deb-simple;  go build -o drone-deb-simple

FROM alpine

COPY --from=builder /go/drone-deb-simple/drone-deb-simple /bin/

RUN apk -Uuv add ca-certificates

ENTRYPOINT ["/bin/drone-deb-simple"]