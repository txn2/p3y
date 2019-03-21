FROM alpine:3.9

COPY p3y /

WORKDIR /

ENTRYPOINT ["/p3y"]