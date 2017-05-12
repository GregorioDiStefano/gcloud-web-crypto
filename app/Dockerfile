FROM alpine:3.5
COPY app /
RUN apk add --update ca-certificates
RUN chmod +x /app
EXPOSE 3000 3000
ENTRYPOINT ["/app"]
