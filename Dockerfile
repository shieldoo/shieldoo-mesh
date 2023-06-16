FROM alpine:3.18

WORKDIR /app

COPY out/asset/linux-amd64/shieldoo-mesh-srv /app/shieldoo-mesh-srv
RUN chmod 550 /app/shieldoo-mesh-srv

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache libc6-compat gcompat

WORKDIR /app
COPY start.sh ./
RUN chmod 550 /app/start.sh

ENTRYPOINT ["/app/start.sh"]