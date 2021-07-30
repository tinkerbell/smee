FROM alpine:3.14
EXPOSE 67 69

RUN apk add --update --upgrade --no-cache net-tools dhclient dhcpcd tftp-hpa curl tcpdump

COPY test-boots.sh /test-boots.sh

ENTRYPOINT /test-boots.sh
