FROM scratch
LABEL org.opencontainers.image.title="platform-contracts"
COPY schema/ /schema/
COPY go.mod /go.mod
COPY gen/go/events/*.go /gen/go/events/
COPY gen/go/internalhttp/*.go /gen/go/internalhttp/
COPY rust/Cargo.toml rust/Cargo.lock /rust/
COPY rust/src/*.rs /rust/src/
