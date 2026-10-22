.PHONY: lint format generate clean

lint:
	buf lint

format:
	buf format -w

generate:
	buf generate

clean:
	rm -rf gen/
