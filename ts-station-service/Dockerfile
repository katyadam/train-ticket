FROM golang:1.22.12-alpine3.21 AS build
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ts-station-service ./cmd/server

FROM alpine:3.21.3
RUN addgroup -S app \
    && adduser -S -G app app
ENV TZ=Asia/Shanghai
COPY --from=build /out/ts-station-service /app/ts-station-service
USER app
EXPOSE 12345
ENTRYPOINT ["/app/ts-station-service"]
