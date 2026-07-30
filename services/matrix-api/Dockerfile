FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/matrix-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/matrix-api /matrix-api
EXPOSE 8080
ENTRYPOINT ["/matrix-api"]
