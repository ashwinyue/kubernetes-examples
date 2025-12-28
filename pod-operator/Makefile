.PHONY: all build install test run deploy undeploy

# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# Run tests
test: go-test
go-test: fmt vet
	go test ./... -coverprofile coverage.out

##@ Build

build: fmt
	go build -o bin/manager main.go

fmt:
	go fmt ./...

vet:
	go vet ./...

##@ Deployment

install:
	kubectl apply -f config/rbac/serviceaccount.yaml
	kubectl apply -f config/rbac/podmanager_controller_role.yaml
	kubectl apply -f config/rbac/rolebinding.yaml
	kubectl apply -f config/crd/bases/apps.mycompany.com_podmanagers.yaml

deploy:
	kubectl apply -f config/manager/manager.yaml

undeploy:
	kubectl delete -f config/manager/manager.yaml
	kubectl delete -f config/rbac/rolebinding.yaml
	kubectl delete -f config/rbac/podmanager_controller_role.yaml
	kubectl delete -f config/rbac/serviceaccount.yaml
	kubectl delete -f config/crd/bases/apps.mycompany.com_podmanagers.yaml

##@ Run

run: fmt
	go run main.go

##@ Clean

clean:
	rm -rf bin/
