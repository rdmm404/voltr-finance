SHELL := /bin/sh
TAILWIND := .tools/tailwindcss
CSS_INPUT := internal/webui/assets/input.css
CSS_OUTPUT := internal/webui/assets/dist/app.css

.PHONY: generate templ tailwind-install css css-watch dev verify-generated

generate: templ css

templ:
	go tool templ generate

tailwind-install:
	./scripts/install-tailwind.sh

$(TAILWIND): scripts/install-tailwind.sh
	./scripts/install-tailwind.sh

css: $(TAILWIND)
	$(TAILWIND) -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify

css-watch: $(TAILWIND)
	$(TAILWIND) -i $(CSS_INPUT) -o $(CSS_OUTPUT) --watch

dev: $(TAILWIND)
	@trap 'kill 0' INT TERM EXIT; \
	go tool templ generate --watch & \
	$(MAKE) css-watch & \
	air & \
	wait

verify-generated: generate
	git diff --exit-code -- '*_templ.go'
	git diff --exit-code -- $(CSS_OUTPUT)
