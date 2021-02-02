GIT_VER := $(shell git describe --tags)
export GO111MODULE := on

.PHONY: test binary install clean

ecs_instance_replace: *.go cmd/ecs_instance_replace/*.go go.*
	go build -trimpath -ldflags "-s -w -X main.Version=${GIT_VER}" -o ecs_instance_replace cmd/ecs_instance_replace/main.go

test:
	go test -v .

# https://goreleaser.com/install/#running-with-docker
goreleaser/build:
	docker run --rm --privileged \
		-v ${PWD}:/go/src/github.com/user/repo \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/github.com/user/repo \
		goreleaser/goreleaser \
		build \
		--rm-dist --skip-validate --snapshot

clean:
	rm -f ecs_instance_replace
