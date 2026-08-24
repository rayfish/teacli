# Build local binary
build:
	go build -o teacli -ldflags="-s -w" .

# Build and install locally (override dir: `just install /usr/local/bin`)
install dir="~/.local/bin":
	go build -o teacli -ldflags="-s -w" .
	@dir=$(eval echo "{{dir}}"); \
	mkdir -p "$dir" 2>/dev/null || true; \
	if [ -w "$dir" ]; then mv teacli "$dir/teacli"; else echo "Installing to $dir (requires sudo)"; sudo mv teacli "$dir/teacli"; fi; \
	echo "Installed teacli to $dir/teacli"; \
	case ":$PATH:" in *":$dir:"*) ;; *) echo "warning: $dir is not on your PATH" ;; esac; \
	"$dir/teacli" --version || true

# Run tests
test:
	go test ./...

# Everything CI checks, in the same order
check:
	@unformatted=$(gofmt -l .); \
	if [ -n "$unformatted" ]; then echo "these files need gofmt:" >&2; echo "$unformatted" >&2; exit 1; fi
	go vet ./...
	go test ./...
	go build ./...

# Clean build artifacts
clean:
	rm -f teacli
	rm -rf dist/

# Build multi-platform release binaries
release:
	@echo "Building release binaries..."
	mkdir -p dist
	
	GOOS=linux GOARCH=amd64 go build -o dist/teacli-linux-amd64 -ldflags="-s -w" .
	GOOS=linux GOARCH=arm64 go build -o dist/teacli-linux-arm64 -ldflags="-s -w" .
	GOOS=darwin GOARCH=amd64 go build -o dist/teacli-darwin-amd64 -ldflags="-s -w" .
	GOOS=darwin GOARCH=arm64 go build -o dist/teacli-darwin-arm64 -ldflags="-s -w" .
	GOOS=windows GOARCH=amd64 go build -o dist/teacli-windows-amd64.exe -ldflags="-s -w" .
	
	chmod +x dist/teacli-linux-* dist/teacli-darwin-*
	@echo "Binaries built in dist/"

# Create and push a new release tag: `just release-tag 1.0.0`
# Tolerates `version=1.0.0` and `v1.0.0`, which just passes through verbatim
# and would otherwise produce a tag named vversion=1.0.0.
release-tag version:
	@v="{{version}}"; v="${v#version=}"; v="${v#v}"; \
	case "$v" in \
		[0-9]*.[0-9]*.[0-9]*) ;; \
		*) echo "expected a version like 1.0.0, got '{{version}}'" >&2; exit 2 ;; \
	esac; \
	if [ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then echo "refusing to tag: not on main" >&2; exit 2; fi; \
	if ! git diff --quiet HEAD; then echo "refusing to tag: working tree is dirty" >&2; exit 2; fi; \
	if ! grep -q "Version:       \"$v\"" cmd/root.go; then echo "refusing to tag: cmd/root.go Version is not $v" >&2; exit 2; fi; \
	git tag "v$v" && git push origin "v$v" && echo "Created and pushed tag v$v"

# Generate checksums for release artifacts
checksums:
	cd dist && sha256sum * > ../checksums.txt && cd ..
	@echo "Checksums generated in checksums.txt"
