PKGS = $(shell go list ./... | grep -v /vendor/)

vet:
	go vet $(PKGS)

fmt:
	go fmt $(PKGS)

update-deps:
	godep save $(PKGS)

.PHONY: vet fmt build update-deps
