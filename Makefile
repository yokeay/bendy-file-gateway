.PHONY: all build build-wasm build-web dev test clean deploy-cf deploy-vercel

# Build everything
all: build-wasm build-web

# Build Go WASM module
build-wasm:
	bash scripts/build-wasm.sh

# Build React admin dashboard
build-web:
	bash scripts/build-web.sh

# Build both
build: build-wasm build-web

# Start web dev server
dev:
	cd web && npm run dev

# Run Go tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf platforms/wasm
	rm -rf platforms/cloudflare/wasm platforms/cloudflare/public
	rm -rf platforms/vercel/wasm platforms/vercel/public
	rm -rf web/dist
	rm -rf web/node_modules

# Install dependencies
deps:
	cd web && npm install

# Deploy to Cloudflare Workers
deploy-cf: build-wasm build-web
	npx wrangler deploy --config platforms/cloudflare/wrangler.toml

# Deploy to Vercel
deploy-vercel: build-wasm build-web
	cd platforms/vercel && vercel deploy --prod

# Initialize the project (first time setup)
init: deps build
	@echo "Project initialized successfully."
	@echo "Run 'make dev' to start the development server."
