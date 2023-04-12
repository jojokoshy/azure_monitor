# Get repo for Azure Monitor MI
FROM bitnami/git AS git-az_monitor_mi
WORKDIR /app
RUN git clone --depth 2  https://github.com/jojokoshy/azure_monitor azure_monitor_mi

# Build Azure Monitor MI binary

FROM golang:1.16-alpine AS binary-az_monitor_mi
WORKDIR /go/src/app/
COPY --from=git-az_monitor_mi /app/ ./
WORKDIR /go/src/app/azure_monitor_mi
RUN go build -o azure_monitor_mi cmd/main.go

# Get repo for Random Int Generator Telegraf Plugin
FROM bitnami/git AS git-random_int_generator
WORKDIR /app
RUN git clone --depth 2  https://github.com/ssoroka/rand.git rand

# Build Random Int Generator Telegraf Plugin binary
FROM golang:1.16-alpine AS binary-rand
WORKDIR /go/src/app/
COPY --from=git-random_int_generator /app/ ./
WORKDIR /go/src/app/rand
RUN go build -o rand cmd/main.go

# Build the telegraf container with your plugins
FROM telegraf:alpine
COPY --from=binary-az_monitor_mi /go/src/app/azure_monitor_mi/azure_monitor_mi /usr/local/bin/azure_monitor_mi
COPY --from=binary-rand /go/src/app/rand/rand /usr/local/bin/rand