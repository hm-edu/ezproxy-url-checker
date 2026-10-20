FROM golang:1.27-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY main.go ./
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
