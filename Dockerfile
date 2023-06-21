FROM golang:1.20 AS builder

COPY . /src
WORKDIR /src

RUN go mod download
RUN mkdir /out
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /out ./...

FROM scratch
WORKDIR /app

COPY --from=builder /out ./
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["./client"]
