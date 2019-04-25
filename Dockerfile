FROM golang:alpine AS builder
RUN apk add --no-cache git make
WORKDIR /go/src/github.com/havuz/havuz
COPY . .
RUN make install

FROM alpine:3.9
COPY --from=builder /go/bin/havuz /
EXPOSE 8080/tcp
CMD ["/havuz", "gateway"]
