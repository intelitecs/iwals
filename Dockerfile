FROM golang:1.18.0-alpine3.15 AS build
#RUN mkdir -p /go/src/iwals
WORKDIR /go/src/iwals
COPY . .
RUN apk add git

RUN CGO_ENABLED=0 go build -o /go/bin/iwals  ./cmd/iwals
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.11 && \
    wget -qO /go/bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /go/bin/grpc_health_probe

FROM scratch
COPY --from=build /go/bin/iwals /bin/iwals
COPY --from=build /go/bin/grpc_health_probe /bin/grpc_health_probe
ENTRYPOINT [ "/bin/iwals" ]