FROM golang:1.24 AS builder

WORKDIR /app

COPY . .

RUN go mod tidy
# These flags ensure Alpine compatibility
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o blockchain-node ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o web-ui ./src/cmd/webui/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/blockchain-node .
COPY --from=builder /app/web-ui .
COPY scripts/start.sh .
COPY src/web ./web

RUN chmod +x start.sh
RUN chmod +x ./blockchain-node ./web-ui