ARCH?=amd64
REPO?=#your repository here
VERSION?=0.1

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o ./bin/prometheus-scraper-generic main.go

container: build
	docker build -t $(REPO)prometheus-scraper-generic:$(VERSION) .
	docker push $(REPO)prometheus-scraper-generic:$(VERSION)
