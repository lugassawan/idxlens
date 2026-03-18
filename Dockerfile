# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /idxlens ./cmd/idxlens

# Stage 2: Final
FROM scratch

COPY --from=builder /idxlens /idxlens

ENTRYPOINT ["/idxlens"]
