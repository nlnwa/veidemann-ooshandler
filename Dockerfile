FROM golang:alpine as builder

WORKDIR /go/src/github.com/nlnwa/veidemann-ooshandler
COPY . .

# Compile the binary statically, so it can be run without libraries.
RUN CGO_ENABLED=0 GOOS=linux go test ./... -a -ldflags '-extldflags "-s -w -static"' .
RUN CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-s -w -static"' .

FROM scratch
COPY --from=builder /go/bin/veidemann-ooshandler /usr/local/bin/veidemann-ooshandler

EXPOSE 9301 50052
VOLUME "/data"

ENTRYPOINT ["/usr/local/bin/veidemann-ooshandler"]
