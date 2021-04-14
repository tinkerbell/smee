FROM alpine:3.12

ARG TARGETARCH
ARG TARGETVARIANT

ENTRYPOINT ["/usr/bin/boots"]
EXPOSE 67 69 80

RUN apk add --update --upgrade --no-cache ca-certificates socat
COPY cmd/boots/boots-linux-${TARGETARCH:-amd64}${TARGETVARIANT} /usr/bin/boots
