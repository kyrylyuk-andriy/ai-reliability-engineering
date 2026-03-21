IMAGE ?= ghcr.io/kyrylyuk-andriy/k8s-health-checker
TAG ?= $(shell git describe --tags --always --dirty)

.PHONY: build test lint docker-build docker-push

build:
	go build -o bin/k8s-health-checker .

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

docker-push: docker-build
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest
