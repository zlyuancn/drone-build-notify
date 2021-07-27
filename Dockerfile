FROM golang as builder
ENV GOPROXY='https://goproxy.cn,https://goproxy.io,direct' CGO_ENABLED=0
ADD . /src
WORKDIR /src
RUN go build .

FROM alpine
EXPOSE 80
ADD conf/ /app/conf
COPY --from=builder /src/drone-build-notify /app/

WORKDIR /app/
CMD ./drone-build-notify
