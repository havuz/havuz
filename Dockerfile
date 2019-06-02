FROM golang:alpine AS builder
RUN apk add --no-cache git make upx
WORKDIR /go/src/github.com/havuz/havuz
COPY . .
RUN make install
RUN upx -q /go/bin/havuz

FROM alpine:3.9
COPY --from=builder /go/bin/havuz /
EXPOSE 8080/tcp
CMD ["/havuz", "gateway"]
