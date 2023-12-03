FROM golang:alpine AS builder
WORKDIR /stahl
COPY  . .
RUN apk update
RUN CGO_ENABLED=0 GOOS=linux go build -o main -ldflags="-s -w" -a -installsuffix cgo ./cmd/stahl/main.go

FROM alpine
WORKDIR /stahl
COPY --from=builder ./stahl .

EXPOSE 8000


ENTRYPOINT ["./main"]