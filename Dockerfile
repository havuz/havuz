FROM golang:alpine AS builder
WORKDIR /go/src/github.com/havuz/havuz
COPY . .
RUN go install -ldflags="-w -s" ./...

FROM alpine
COPY --from=builder /go/bin/havuz /
EXPOSE 8080/tcp
CMD ["/havuz", "gateway"]