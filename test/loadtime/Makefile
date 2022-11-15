OUTPUT?=build/

build:
	go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' -o $(OUTPUT)load ./cmd/load/
	go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' -o $(OUTPUT)report ./cmd/report/
.PHONY: build
