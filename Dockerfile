FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o kubefirst-api .

FROM alpine:3.18.2

RUN apk update && \
    apk add --no-cache \
        git \
        openssh \
        aws-cli \
        curl

RUN mkdir -p /root/.ssh \
    && ssh-keyscan github.com >> /root/.ssh/known_hosts \
    && ssh-keyscan gitlab.com >> /root/.ssh/known_hosts

COPY --from=builder /build/kubefirst-api /
COPY --from=builder /build/docs /docs

EXPOSE 8081

ENTRYPOINT ["/kubefirst-api"]
