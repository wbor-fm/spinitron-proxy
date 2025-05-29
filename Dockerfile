FROM golang:1.22-alpine as base

WORKDIR /app

COPY . .

RUN go build -o /main

FROM golang:1.22-alpine

COPY --from=base /main /main

# Docker health check
# Checks every 30 seconds, times out after 5 seconds.
# Gives the container 10 seconds to start before first check.
# Marks unhealthy after 3 consecutive failures.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/healthz || exit 1

EXPOSE 8080

CMD [ "/main" ]
