.PHONY: build run-ui fetch-btc backtest optimize parse-strategy test vet tidy docker-up docker-down clean

build:
	go build -o bin/server ./cmd/server
	go build -o bin/fetch ./cmd/fetch
	go build -o bin/backtest ./cmd/backtest
	go build -o bin/optimize ./cmd/optimize
	go build -o bin/parse-strategy ./cmd/parse-strategy

run-ui:
	go run ./cmd/server

fetch-btc:
	go run ./cmd/fetch --run configs/runs/btc_1h_ema.yaml

backtest:
	go run ./cmd/backtest --run configs/runs/btc_1h_ema.yaml

optimize:
	go run ./cmd/optimize --run configs/runs/btc_1h_ema.yaml

parse-strategy:
	go run ./cmd/parse-strategy --in strategies/example.pine --out configs/strategies/example.generated.yaml --name example_ema

fetch:
	go run ./cmd/fetch --symbol $(SYMBOL) --timeframe $(TF) --from $(FROM) --to $(TO)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/ data/ results/
