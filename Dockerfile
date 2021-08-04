# run `make image` to build the binary + container
# if you're using `make boots` this Dockerfile will not find the binary
# and you probably want `make cmd/boots/boots-linux-amd64`
FROM alpine:3.13

ARG TARGETARCH
ARG TARGETVARIANT

ENTRYPOINT ["/usr/bin/boots"]
EXPOSE 67 69 80

RUN apk add --update --upgrade --no-cache ca-certificates socat
COPY cmd/boots/boots-linux-${TARGETARCH:-amd64}${TARGETVARIANT} /usr/bin/boots
