BIN_DIR := bin
CMDS := edgesync faultctl bench checker
.PHONY: all build test race vet fmt bench serve cluster-up cluster-down cluster-status loadtest durability docker compose-up clean
all: fmt vet test build
build:
	@mkdir -p $(BIN_DIR)
	@for c in $(CMDS); do go build -o $(BIN_DIR)/$$c ./cmd/$$c || exit 1; done
test:
	go test ./...
race:
	go test -race -count=1 ./...
vet:
	go vet ./...
fmt:
	gofmt -l -w .
bench:
	go test -run '^$$' -bench . -benchmem ./pkg/...
serve: build
	./$(BIN_DIR)/edgesync -http :8080 -data ./data/solo -seed
cluster-up: build
	./scripts/cluster.sh start
cluster-down:
	./scripts/cluster.sh stop
cluster-status: build
	./$(BIN_DIR)/faultctl status
loadtest: build
	./$(BIN_DIR)/bench -addrs localhost:8081,localhost:8082,localhost:8083 -clients 32 -duration 20s -csv results.csv -label local3
durability: build
	./$(BIN_DIR)/checker -duration 40s
docker:
	docker build -t edgesync:dev .
compose-up:
	docker compose up --build
clean:
	rm -rf $(BIN_DIR) data results.csv
