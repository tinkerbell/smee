FROM alpine:3.14
EXPOSE 67 69

RUN apk add --update --upgrade --no-cache net-tools busybox tftp-hpa curl tcpdump

COPY busybox-udhcpc-script.sh /busybox-udhcpc-script.sh
COPY extract-traceparent-from-opt43.sh /extract-traceparent-from-opt43.sh
COPY test-boots.sh /test-boots.sh

ENTRYPOINT /test-boots.sh
