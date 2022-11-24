FROM alpine:3.15

WORKDIR /app

COPY out/asset/linux-amd64/shieldoo-mesh-srv /app/shieldoo-mesh-srv
RUN chmod 550 /app/shieldoo-mesh-srv

COPY install/linux/wstunnel-amd64 /app/wstunnel
RUN chmod 550 /app/wstunnel
RUN ln -s /app/wstunnel /lib/wstunnel

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache libc6-compat gcompat

WORKDIR /app

ENTRYPOINT ["/app/shieldoo-mesh-srv", "-run"]