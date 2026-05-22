.PHONY: generate-mocks test-unit test-functional

generate-mocks:
	mockery --dir=internal --all --output=test/mocks --outpkg=mocks

test-unit:
	go test ./test/unit/... -count=1 -timeout 30s -coverprofile=coverage.out

test-functional:
	go test ./test/functional/... -count=1 -timeout 120s -v
