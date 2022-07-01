FROM golang:1.18-bullseye as builder
RUN go install golang.org/dl/go1.18@latest \
    && go1.18 download

WORKDIR /build


COPY server/ ./server
COPY proto/ ./proto

RUN ls -l

WORKDIR /build/server
RUN go mod tidy
RUN ls -l
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go

FROM scratch

COPY --from=builder /build/server /server
ENTRYPOINT ["/server/main"]