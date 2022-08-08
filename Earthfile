VERSION 0.6
FROM alpine

ARG K3S_VERSION
ARG C3OS_BASE_IMAGE=quay.io/c3os/c3os-core:opensuse-latest
ARG IMAGE=quay.io/c3os/provider-k3s:dev


ARG LUET_VERSION=0.32.4
ARG GOLANG_VERSION=1.18

go-deps:
    FROM golang:$GOLANG_VERSION
    WORKDIR /build
    COPY go.mod go.sum ./
    RUN go mod download
    RUN apt-get update && apt-get install -y upx
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum

BUILD_GOLANG:
    COMMAND
    WORKDIR /build
    COPY . ./
    ARG BIN
    ARG SRC

    RUN go build -ldflags "-s -w" -o ${BIN} ./${SRC} && upx ${BIN}
    SAVE ARTIFACT ${BIN} ${BIN} AS LOCAL build/${BIN}

build-provider:
    FROM +go-deps
    DO +BUILD_GOLANG --BIN=agent-provider-k3s --SRC=main.go

lint:
    FROM golang:$GOLANG_VERSION
    RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.46.2
    WORKDIR /build
    COPY . .
    RUN golangci-lint run

docker:
    FROM ${C3OS_BASE_IMAGE}

    ENV INSTALL_K3S_VERSION=${K3S_VERSION}
    ENV INSTALL_K3S_BIN_DIR="/usr/bin"
    ENV K3S_CONFIG_FILE="/etc/rancher/k3s/config.d"
    RUN curl -sfL https://get.k3s.io > installer.sh \
        INSTALL_K3S_SKIP_START="true" INSTALL_K3S_SKIP_ENABLE="true" bash installer.sh \
        INSTALL_K3S_SKIP_START="true" INSTALL_K3S_SKIP_ENABLE="true" bash installer.sh agent \
        rm -rf installer.sh

    COPY +build-provider/agent-provider-k3s /system/providers/agent-provider-k3s

    SAVE IMAGE --push $IMAGE

