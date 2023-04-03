FROM golang:1.20 AS build

WORKDIR /src
COPY ["*.go", "go.mod", "go.sum", "./"]
COPY goproxy ./goproxy
COPY test ./test
COPY wstunnel ./wstunnel
COPY rpc ./rpc

RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc libgtk-3-dev libappindicator3-dev && \
    apt-get install -y --no-install-recommends nsis nsis-doc nsis-pluginapi 

RUN env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-srv

FROM alpine:3.15

WORKDIR /app

COPY --from=build /src/out/shieldoo-mesh-srv /app/shieldoo-mesh-srv
RUN chmod 550 /app/shieldoo-mesh-srv

COPY install/linux/wstunnel-amd64 /app/wstunnel
RUN chmod 550 /app/wstunnel
RUN ln -s /app/wstunnel /lib/wstunnel

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache libc6-compat gcompat

WORKDIR /app
COPY start.sh ./

ENTRYPOINT ["/app/start.sh"]