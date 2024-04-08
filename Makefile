all:
	go build

.PHONY: spdx
spdx:
	./tools/spdx-ensure 
