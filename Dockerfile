FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/web

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/server ./server
COPY web/ ./web/
COPY words.txt words-common.txt words-large.txt ./
COPY favico.ico ./

EXPOSE 8080

CMD ["./server"]
