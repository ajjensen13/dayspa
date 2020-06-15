ARG GO_VERSION=1
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as build
ARG GCFLAGS="-c 1"
ENV CGO_ENABLED=0
ENV GO111MODULE=on
WORKDIR /src/app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN go test -json ./...
RUN go build -gcflags "$GCFLAGS" -o /bin/dayspa

FROM alpine:${ALPINE_VERSION} as run
WORKDIR /src/app
COPY --from=build /bin/dayspa /bin/dayspa
ENTRYPOINT [ "/bin/dayspa", "--mode=ngsw", "--webroot=/var/www/html" ]