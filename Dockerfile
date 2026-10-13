FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN mkdir -p /out /data && CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/edgesync ./cmd/edgesync

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/edgesync /usr/local/bin/edgesync
COPY --from=build --chown=65532:65532 /data /data
COPY --from=build /src/web/index.html /app/web/index.html
USER nonroot:nonroot
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/edgesync"]
