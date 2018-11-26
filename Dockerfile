FROM golang:1.11-stretch
WORKDIR /go/src/app
COPY . .
RUN go build -o /usr/local/bin/http-bench-target

CMD ["/usr/local/bin/http-bench-target"]
