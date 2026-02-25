.PHONY: all build windows embed-latest test clean bench reset-nnue shuffler converter train sprt

# Binary name
BINARY_NAME=axon

# Paths
BULLET_DIR=../bullet
EMBED_PATH=internal/nnue/embedded.nnue

all: build

build:
	go build -o $(BINARY_NAME) main.go

shuffler:
	go build -o shuffler cmd/shuffler/main.go

converter:
	go build -o converter cmd/converter/main.go

sprt:
	go build -o sprt cmd/sprt/main.go

train:
	cd $(BULLET_DIR) && cargo run --release --example axon --no-default-features --features cpu

# Cross-compile for Windows
windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe main.go

# Finds the latest completed checkpoint in the bullet directory and embeds it
embed-latest:
	@echo "Searching for latest completed checkpoint in $(BULLET_DIR)/checkpoints..."
	@LATEST_DIR=$$(find $(BULLET_DIR)/checkpoints -name "quantised.nnue" -printf '%h\n' | sort -V | tail -n 1); \
	if [ -z "$$LATEST_DIR" ]; then \
		echo "Error: No completed checkpoints (quantised.nnue) found."; \
		exit 1; \
	fi; \
	echo "Found latest completed checkpoint: $$LATEST_DIR"; \
	cp "$$LATEST_DIR/quantised.nnue" $(EMBED_PATH); \
	echo "Successfully embedded $$LATEST_DIR/quantised.nnue into $(EMBED_PATH)"

# Runs all tests
test:
	go test -v ./...

# Runs a quick benchmark to verify everything is working
bench: build
	./$(BINARY_NAME) bench 10

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f shuffler
	rm -f converter
	rm -f sprt

# Clears the embedded network weights
reset-nnue:
	truncate -s 0 $(EMBED_PATH)
