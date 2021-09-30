FROM golang:alpine AS builder
WORKDIR /go/src/github.com/wzshiming/sshd/
COPY . .
ENV CGO_ENABLED=0
RUN go install ./cmd/sshd

FROM alpine
EXPOSE 8080
COPY --from=builder /go/bin/sshd /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/sshd" ]
