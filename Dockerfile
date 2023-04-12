FROM bitnami/git AS git-psi
WORKDIR /app
RUN git clone --depth 1 --branch v0.1.0 https://github.com/gridscale/linux-psi-telegraf-plugin.git linux-psi-telegraf-plugin

# Build psi binary

FROM golang:1.16-alpine AS binary-psi
WORKDIR /go/src/app/
COPY --from=git-psi /app/ ./
WORKDIR /go/src/app/linux-psi-telegraf-plugin
RUN go build -o psi cmd/main.go

# Build the telegraf container with your plugins
FROM telegraf:alpine

COPY --from=binary-psi /go/src/app/linux-psi-telegraf-plugin/psi /usr/local/bin/psi