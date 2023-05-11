all:
	go build

.PHONY: lint
lint:
	$(MAKE) -C gotools
	GOOS=linux   ./gotools/golangci-lint run
	GOOS=windows ./gotools/golangci-lint run

.PHONY: spdx
spdx:
	./tools/spdx-ensure 
