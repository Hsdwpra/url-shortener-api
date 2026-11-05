FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy && go build -ldflags='-s -w' -o /shorty .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /shorty /shorty
RUN mkdir -p /app/data
EXPOSE 8080
ENTRYPOINT ["/shorty"]
