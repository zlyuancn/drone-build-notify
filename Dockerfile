FROM alpine
EXPOSE 80
ADD drone-build-notify /app/
ADD conf/ /app/conf

WORKDIR /app/
CMD ["./drone-build-notify"]
