FROM golang:1.22-bookworm as builder
WORKDIR /go/src/app
COPY . .
RUN go build -o /usr/local/bin/http-bench-target

FROM debian:bookworm-slim
COPY --from=builder /usr/local/bin/http-bench-target /usr/local/bin
ENTRYPOINT ["/usr/local/bin/http-bench-target"]
