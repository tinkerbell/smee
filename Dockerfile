FROM alpine:3.11
ENTRYPOINT ["/usr/bin/boots"]
EXPOSE 67 69 80
RUN apk add --update --upgrade --no-cache ca-certificates socat
COPY . /usr/myapp
RUN cp /usr/myapp/boots-linux-$(uname -m) /usr/bin/boots
RUN rm -fr /usr/myapp
