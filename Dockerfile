FROM nixos/nix:2.3.6

COPY . /usr/myapp
WORKDIR /usr/myapp

RUN nix-shell --run 'make boots'

FROM alpine:3.11

ENTRYPOINT ["/usr/bin/boots"]
# ENV GIN_MODE release
EXPOSE 67 69 80

RUN apk add --update --upgrade --no-cache ca-certificates socat
COPY --from=0 /usr/myapp/boots /usr/bin/boots
