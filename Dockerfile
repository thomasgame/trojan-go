FROM golang:1.26-alpine AS builder
WORKDIR /workspace
ARG REF
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY

RUN apk add --no-cache git make wget ca-certificates
COPY go.work go.work.sum ./
COPY src/go.mod src/go.sum ./src/
RUN go -C src mod download
COPY . .
RUN if [ -n "${REF}" ]; then echo "Building local checkout for ${REF}"; fi
RUN make &&\
    wget https://github.com/v2fly/domain-list-community/raw/release/dlc.dat -O build/geosite.dat &&\
    wget https://github.com/v2fly/geoip/raw/release/geoip.dat -O build/geoip.dat &&\
    wget https://github.com/v2fly/geoip/raw/release/geoip-only-cn-private.dat -O build/geoip-only-cn-private.dat

FROM alpine
WORKDIR /
RUN apk add --no-cache tzdata ca-certificates
COPY --from=builder /workspace/build /usr/local/bin/
COPY --from=builder /workspace/example/server.json /etc/trojan-go/config.json

ENTRYPOINT ["/usr/local/bin/trojan-go", "-config"]
CMD ["/etc/trojan-go/config.json"]
