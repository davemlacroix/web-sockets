# syntax=docker/dockerfile:1

FROM golang:1.26 AS base
FROM base AS build

WORKDIR /src

# download go modules
COPY go.mod ./
RUN go mod download

# build
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/websockets

# publish
FROM base AS publish
WORKDIR /app
COPY --from=build /app /app
CMD ["/app/websockets"]
