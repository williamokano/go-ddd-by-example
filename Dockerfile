# One static binary, two subcommands: serve and migrate.
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /stagehand ./cmd/stagehand

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /stagehand /stagehand
EXPOSE 8080
ENTRYPOINT ["/stagehand"]
CMD ["serve"]
