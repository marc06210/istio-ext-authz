FROM golang:1.21.5-alpine as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build *.go

FROM gcr.io/distroless/base
COPY --from=build /app/ext_authz_basic /

EXPOSE 9000 8000

ENTRYPOINT ["/ext_authz_basic"]