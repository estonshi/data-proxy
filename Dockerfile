FROM alpine:3.15.4

COPY data-proxy /data-proxy
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

ENTRYPOINT ["/data-proxy"]
