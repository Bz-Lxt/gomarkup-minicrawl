FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/minicrawl ./cmd/server

FROM golang:1.22-alpine AS test
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["go", "test", "./...", "-count=1", "-timeout", "60s"]

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 65532 crawler
ENV TZ=Asia/Shanghai
COPY --from=build /out/minicrawl /usr/local/bin/minicrawl
USER crawler
EXPOSE 8080
ENTRYPOINT ["minicrawl"]
