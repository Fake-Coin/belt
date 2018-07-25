FROM alpine

RUN apk update
RUN apk add ca-certificates

## Network
EXPOSE 8081

## data/configuration
VOLUME "/mnt/belt"

ADD belt /

CMD ["/belt", "-conf", "/mnt/belt"]
