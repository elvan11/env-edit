FROM golang:1.22-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends mingw-w64 pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

ARG GO_LDFLAGS=-H windowsgui

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
ENV GOOS=windows
ENV GOARCH=amd64
ENV CC=x86_64-w64-mingw32-gcc
ENV CXX=x86_64-w64-mingw32-g++

RUN go build -ldflags="${GO_LDFLAGS}" -o /out/env-edit.exe .

FROM scratch AS artifact

COPY --from=builder /out/env-edit.exe /env-edit.exe
