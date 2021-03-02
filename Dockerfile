FROM golang:1.16-buster as builder
WORKDIR /go/src/app
COPY . .
RUN go build -o /usr/local/bin/http-bench-target

FROM debian:buster-slim
COPY --from=builder /usr/local/bin/http-bench-target /usr/local/bin
ENTRYPOINT ["/usr/local/bin/http-bench-target"]
