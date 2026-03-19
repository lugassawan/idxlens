# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /idxlens ./cmd/idxlens

# Stage 2: Final
FROM alpine:3.21

RUN apk add --no-cache \
    chromium \
    ca-certificates \
    tzdata

# chromedp expects Chrome at a well-known path
ENV CHROME_PATH=/usr/bin/chromium-browser

COPY --from=builder /idxlens /usr/local/bin/idxlens

ENTRYPOINT ["idxlens"]
