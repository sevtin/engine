FROM golang:1.13 as build
RUN mkdir -p /build/matching-engine
WORKDIR /build/matching-engine/

# Force the go compiler to use modules
ENV GO111MODULE=on

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
# COPY go.sum .

# This is the ‘magic’ step that will download all the dependencies that are specified in 
# the go.mod and go.sum file.
# Because of how the layer caching system works in Docker, the  go mod download 
# command will _ only_ be re-run when the go.mod or go.sum file change 
# (or when we add another docker instruction below this line)
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -a -installsuffix cgo --ldflags "-s -w" -o /usr/bin/matching_engine

FROM alpine:3.9

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY --from=build /usr/bin/matching_engine /root/
COPY --from=build /build/matching-engine/.engine.yml /root/

ENV LOG_LEVEL="info"
ENV LOG_FORMAT="json"

EXPOSE 6060
RUN mkdir -p /root/backups
WORKDIR /root/

CMD ["./matching_engine", "--log-level=${LOG_LEVEL}", "--log-format=${LOG_FORMAT}", "server"]