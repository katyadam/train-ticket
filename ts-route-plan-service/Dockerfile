FROM golang:1.22.12-alpine3.21 AS build
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/ts-route-plan-service ./cmd/server

FROM alpine:3.21.3
RUN addgroup -S app \
    && adduser -S -G app app
ENV TZ=Asia/Shanghai
COPY --from=build /out/ts-route-plan-service /app/ts-route-plan-service
USER app
EXPOSE 14578
ENTRYPOINT ["/app/ts-route-plan-service"]
