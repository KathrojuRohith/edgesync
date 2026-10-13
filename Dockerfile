FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/edgesync ./cmd/edgesync \
 && CGO_ENABLED=0 go build -o /out/faultctl ./cmd/faultctl \
 && CGO_ENABLED=0 go build -o /out/bench ./cmd/bench \
 && CGO_ENABLED=0 go build -o /out/checker ./cmd/checker \
 && mkdir /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /usr/local/bin/
COPY --from=build --chown=65532:65532 /data /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/edgesync"]
