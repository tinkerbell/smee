# run `make image` to build the binary + container
# if you're using `make build` this Dockerfile will not find the binary
# and you probably want `make smee-linux-amd64`
FROM alpine:3.21

ARG TARGETARCH
ARG TARGETVARIANT

ENTRYPOINT ["/usr/bin/smee"]
EXPOSE 67 69 80

RUN apk add --update --upgrade --no-cache ca-certificates
COPY cmd/smee/smee-linux-${TARGETARCH:-amd64}${TARGETVARIANT} /usr/bin/smee
