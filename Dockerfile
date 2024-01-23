# Build stage
FROM golang:1.21.5 AS builder

WORKDIR /src
COPY . .

RUN go test --cover -v ./cmd/web
RUN CGO_ENABLED=0 go build -ldflags='-w -s' -v -o web ./cmd/web/

# Image stage
FROM scratch

COPY --from=builder /src/web /usr/local/bin/web

EXPOSE 4000

ENTRYPOINT ["web"]
