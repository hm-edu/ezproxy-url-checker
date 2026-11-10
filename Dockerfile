FROM golang:1.27-alpine@sha256:eb61c3873c97416fff0116553fff65d67a3549f72e11d9758849a689cdd8b523 AS build

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
