FROM golang:alpine AS build

ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./out/executable .


FROM scratch

COPY --from=build /app/out/executable /executable

COPY static ./static
COPY templates ./templates

USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/executable"]
