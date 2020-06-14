FROM alpine

ADD drone-deb-simple /bin/

RUN apk -Uuv add ca-certificates

ENTRYPOINT ["/bin/drone-deb-simple"]