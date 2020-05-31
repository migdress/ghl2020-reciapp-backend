.PHONY: build deploy

build:
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/v1 v1/*.go

deploy: build
	sls deploy --verbose

redeploy: build
	sls remove -v
	sls deploy -v
