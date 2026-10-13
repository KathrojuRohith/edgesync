BIN_DIR := bin
CMDS := edgesync faultctl bench checker
.PHONY: all build test race vet fmt run bench serve cluster-up cluster-down cluster-status loadtest durability docker compose-up clean
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
	gofmt -w $$(find . -name '*.go' -type f)
run: build
	./$(BIN_DIR)/edgesync -http :8080 -data ./data/solo -seed
serve: run
bench: build
	go test -run '^$$' -bench . -benchmem ./pkg/...
cluster-up: build
	bash scripts/cluster.sh start
cluster-down:
	bash scripts/cluster.sh stop
cluster-status: build
	bash scripts/cluster.sh status
loadtest: build
	./$(BIN_DIR)/bench -addr http://localhost:8081 -clients 32 -duration 20s -csv results.csv
durability: build
	./$(BIN_DIR)/checker -duration 40s
docker:
	docker build -t edgesync:dev .
compose-up:
	docker compose up --build
clean:
	rm -rf $(BIN_DIR) data results.csv
