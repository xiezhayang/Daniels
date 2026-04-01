# syntax=docker/dockerfile:1

FROM golang:1.25.5-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /daniels .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /daniels /daniels

USER nonroot:nonroot

EXPOSE 18080

ENV GATEWAY_PORT=18080

ENTRYPOINT ["/daniels"]
