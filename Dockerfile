FROM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags="-s -w -buildid=" \
  -o /out/ezproxy-url-checker \
  .

FROM scratch

COPY --from=build /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=build /out/ezproxy-url-checker /app/ezproxy-url-checker

WORKDIR /app
USER 1000:1000
EXPOSE 8888

ENTRYPOINT ["/app/ezproxy-url-checker"]
