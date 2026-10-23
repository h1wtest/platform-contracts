FROM scratch
LABEL org.opencontainers.image.title="platform-contracts"
COPY LICENSE /LICENSE
COPY schema/ /schema/
COPY go.mod /go.mod
COPY gen/ /gen/
COPY openapi/ /openapi/
COPY release-manifest.json /release-manifest.json
COPY rust/Cargo.toml rust/Cargo.lock /rust/
COPY rust/src/*.rs /rust/src/
