FROM docker.io/golang:1.21.5 as builder

WORKDIR /go/src/github.com/nlnwa/veidemann-ooshandler

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build  -trimpath -ldflags "-s -w"


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /go/src/github.com/nlnwa/veidemann-ooshandler/veidemann-ooshandler /veidemann-ooshandler

EXPOSE 9301 50052
VOLUME "/data"

ENTRYPOINT ["/veidemann-ooshandler"]
