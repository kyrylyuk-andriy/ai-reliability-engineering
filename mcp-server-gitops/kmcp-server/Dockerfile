FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /k8s-health-checker .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /k8s-health-checker /k8s-health-checker
USER nonroot:nonroot
ENTRYPOINT ["/k8s-health-checker"]
