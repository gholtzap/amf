
FROM golang:1.21-alpine AS builder


RUN apk add --no-cache git make


WORKDIR /app


COPY go.mod go.sum* ./


RUN go mod download


COPY . .


RUN make build


FROM alpine:latest


RUN apk add --no-cache ca-certificates


WORKDIR /app


COPY --from=builder /app/bin/amf /app/amf


COPY --from=builder /app/config /app/config



EXPOSE 8000

EXPOSE 38412


ENTRYPOINT ["/app/amf"]


CMD ["-config", "config/amfcfg.json"]
