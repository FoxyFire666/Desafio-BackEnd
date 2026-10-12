FROM golang:1.22.7 AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/desafio-backend ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /bin/desafio-backend /usr/local/bin/desafio-backend
ENTRYPOINT ["/usr/local/bin/desafio-backend"]
