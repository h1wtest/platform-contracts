PYTHON ?= uv run --no-project python
shell_quote = '$(subst ','"'"',$(1))'
.PHONY: generate test compatibility

generate:
	$(PYTHON) scripts/generate.py

test:
	go test ./...
	cargo test --manifest-path rust/Cargo.toml
	$(PYTHON) -m unittest discover -s tests

compatibility:
	@if [ -z $(call shell_quote,$(strip $(BASE_SCHEMA))) ]; then printf '%s\n' 'BASE_SCHEMA is required: make compatibility BASE_SCHEMA=/path/to/previous/schema.json' >&2; exit 2; fi
	$(PYTHON) scripts/check_compatibility.py $(call shell_quote,$(BASE_SCHEMA)) $(call shell_quote,$(or $(CURRENT_SCHEMA),schema/events-v1.json))
