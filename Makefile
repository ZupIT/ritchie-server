VERSION = $(RELEASE_VERSION)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=ritchie-server
CMD_PATH=./server/cmd/server/main.go

GIT_REMOTE=https://$(GIT_USERNAME):$(GIT_PASSWORD)@github.com/ZupIT/ritchie-server

# Docker
DOCKERCMD=docker
DOCKERBUILD=${DOCKERCMD} build
DOCKERPUSH=${DOCKERCMD} push
DOCKERTAG=${DOCKERCMD} tag
DOCKERLOGIN=${DOCKERCMD} login

GONNA_RELEASE=$(shell ./.circleci/scripts/gonna_release.sh)
NEXT_VERSION=$(shell ./.circleci/scripts/next-version.sh)
IS_RELEASE=$(shell echo $(VERSION) | egrep "^[0-9.]+-beta.[0-9]+")

BUCKET="ritchie-cli-bucket152849730126474"

all: test build

build-local-mac:
	GOOS=darwin GOARCH=amd64 ${GOBUILD} -o ./${BINARY_NAME} -v ${CMD_PATH}

build-local:
	${GOBUILD} -o ./${BINARY_NAME} -v ${CMD_PATH}

delivery-hub:
	echo "${DOCKERHUB_PASS}" | ${DOCKERLOGIN} --username ${DOCKERHUB_USERNAME} --password-stdin
	${DOCKERPUSH} "${DOCKERHUB_USERNAME}/${BINARY_NAME}:${VERSION}"

test:
	DOCKER_REGISTRY_BUILDER= docker-compose -f docker-compose-ci.yml run server

test-local:
	docker-compose up -d
	./.circleci/scripts/run-tests.sh
	docker-compose down

release:
	git config --global user.email "$(GIT_EMAIL)"
	git config --global user.name "$(GIT_NAME)"
	git tag -a $(RELEASE_VERSION) -m "CHANGELOG: https://github.com/ZupIT/ritchie-server/blob/master/CHANGELOG.md"
	git push $(GIT_REMOTE) $(RELEASE_VERSION)
	gem install github_changelog_generator
	github_changelog_generator -u zupit -p ritchie-server --token $(GIT_PASSWORD) --enhancement-labels feature,Feature --exclude-labels duplicate,question,invalid,wontfix
	git add .
	git commit --allow-empty -m "[ci skip] release"
	git push $(GIT_REMOTE) HEAD:release-$(RELEASE_VERSION)
	curl --user $(GIT_USERNAME):$(GIT_PASSWORD) -X POST https://api.github.com/repos/ZupIT/ritchie-server/pulls -H 'Content-Type: application/json' -d '{ "title": "Release $(RELEASE_VERSION) merge", "body": "Release $(RELEASE_VERSION) merge with master", "head": "release-$(RELEASE_VERSION)", "base": "master" }'


delivery-file:
ifneq "$(IS_RELEASE)" ""
	echo -n "$(NEXT_VERSION)" > stable-server.txt
	aws s3 sync . s3://$(BUCKET)/ --exclude "*" --include "stable-server.txt"
else
	echo "NOT GONNA PUBLISH"
endif

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${GOBUILD} -o ./bin/${BINARY_NAME} -v ${CMD_PATH}

build-container:
	cp bin/$(BINARY_NAME) server
	${DOCKERBUILD} -t "${DOCKERHUB_USERNAME}/${BINARY_NAME}:${VERSION}" ./server

release-creator:
ifeq "$(GONNA_RELEASE)" "RELEASE"
	git config --global user.email "$(GIT_EMAIL)"
	git config --global user.name "$(GIT_NAME)"
	git checkout -b "release-$(NEXT_VERSION)"
	git add .
	git commit --allow-empty -m "release-$(NEXT_VERSION)"
	git push $(GIT_REMOTE) HEAD:release-$(NEXT_VERSION)

else
	@echo "NOT GONNA RELEASE"
endif

rebase-beta:
	git config --global user.email "$(GIT_EMAIL)"
	git config --global user.name "$(GIT_NAME)"
	git push $(GIT_REMOTE) --delete beta | true
	git checkout -b beta
	git reset --hard master
	git add .
	git commit --allow-empty -m "beta"
	git push $(GIT_REMOTE) HEAD:beta