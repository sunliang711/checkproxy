FROM golang:1.20-alpine as builder
COPY . /tmp/myService
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn"
WORKDIR /tmp/myService
RUN --mount=type=cache,target=/root/.cache/go-build go build -o checkproxy main.go

FROM alpine
WORKDIR /usr/local/bin

COPY --from=builder /tmp/myService/checkproxy /usr/local/bin/
COPY --from=builder /tmp/myService/config.toml /usr/local/bin/

ENTRYPOINT ["checkproxy"]