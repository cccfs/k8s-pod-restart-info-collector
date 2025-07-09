FROM golang:1.23.4-alpine3.21 AS builder
COPY go.* /
RUN go mod download
COPY *.go /
RUN CGO_ENABLED=0 go build -o /k8s-pod-restart-info-collector /

FROM alpine:3.21
COPY --from=builder /k8s-pod-restart-info-collector /k8s-pod-restart-info-collector
CMD ["/k8s-pod-restart-info-collector"]