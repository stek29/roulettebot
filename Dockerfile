FROM golang:latest AS builder
LABEL org.opencontainers.image.source https://github.com/stek29/roulettebot

ENV GO111MODULE=on

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOBIN=/app/bin go install ./roulette

FROM scratch
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ./i18n ./
COPY --from=builder /app/bin /bin

ENTRYPOINT [ "/bin/roulette" ]
