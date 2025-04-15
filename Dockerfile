FROM golang:alpine AS build

ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./out/executable .


FROM alpine:latest

RUN apk update && \
    apk upgrade --no-cache

RUN addgroup application-group --gid 1001 && \
    adduser application-user --uid 1001 \
        --ingroup application-group \
        --disabled-password

WORKDIR /app

COPY --from=build /app/out .

RUN chown --recursive application-user .
USER application-user

EXPOSE 8080

COPY static ./static
COPY templates ./templates

ENTRYPOINT ["./executable"]
