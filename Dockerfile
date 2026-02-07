FROM golang:1.25.6-alpine3.22 AS builder

WORKDIR /go/src/app
COPY . .
RUN env CGO_ENABLED=0 go build -o /sentinel

FROM scratch AS build-image
COPY --from=builder /sentinel /sentinel
COPY sentinel.yaml /etc/sentinel/sentinel.yaml
ENTRYPOINT ["/sentinel"]
CMD ["start"]