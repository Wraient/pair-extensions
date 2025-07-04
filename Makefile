# Makefile for Extension Development and Testing

# Variables
TESTER_BINARY := extension-tester
EXTENSION_PATH := .
GO_FILES := $(shell find . -name "*.go" -not -path "./test-extension.go")

# Default target
.PHONY: help
help:
	@echo "Extension Development Makefile"
	@echo "=============================="
	@echo ""
	@echo "Available targets:"
	@echo "  build-tester    Build the extension testing tool"
	@echo "  test           Test extension in current directory"
	@echo "  test-verbose   Test with verbose output"
	@echo "  test-detailed  Test with detailed report"
	@echo "  test-json      Test with JSON output"
	@echo "  clean          Remove built binaries"
	@echo "  watch          Watch for changes and auto-test"
	@echo ""
	@echo "Extension-specific targets:"
	@echo "  test-allanime  Test the allanime extension"
	@echo ""
	@echo "Variables:"
	@echo "  EXTENSION_PATH Path to extension (default: .)"
	@echo ""
	@echo "Examples:"
	@echo "  make test EXTENSION_PATH=./src/allanime"
	@echo "  make test-verbose"
	@echo "  make test-json > results.json"

# Build the extension tester
.PHONY: build-tester
build-tester:
	@echo "🔨 Building extension tester..."
	go build -o $(TESTER_BINARY) test-extension.go
	@echo "✅ Extension tester built: $(TESTER_BINARY)"

# Test extension with default settings
.PHONY: test
test: build-tester
	@echo "🧪 Testing extension at: $(EXTENSION_PATH)"
	./$(TESTER_BINARY) -path $(EXTENSION_PATH)

# Test with verbose output
.PHONY: test-verbose
test-verbose: build-tester
	@echo "🧪 Testing extension with verbose output: $(EXTENSION_PATH)"
	./$(TESTER_BINARY) -path $(EXTENSION_PATH) -verbose

# Test with detailed report
.PHONY: test-detailed
test-detailed: build-tester
	@echo "🧪 Testing extension with detailed report: $(EXTENSION_PATH)"
	./$(TESTER_BINARY) -path $(EXTENSION_PATH) -format detailed

# Test with JSON output
.PHONY: test-json
test-json: build-tester
	@./$(TESTER_BINARY) -path $(EXTENSION_PATH) -format json

# Test specific extensions
.PHONY: test-allanime
test-allanime: build-tester
	@echo "🧪 Testing AllAnime extension..."
	./$(TESTER_BINARY) -path ./src/allanime -verbose

# Clean built binaries
.PHONY: clean
clean:
	@echo "🧹 Cleaning built binaries..."
	rm -f $(TESTER_BINARY)
	find . -name "*-test" -type f -delete
	find . -name "*-test.exe" -type f -delete
	@echo "✅ Cleanup complete"

# Watch for changes and auto-test (requires entr: apt install entr)
.PHONY: watch
watch: build-tester
	@echo "👀 Watching for changes in: $(EXTENSION_PATH)"
	@echo "Press Ctrl+C to stop watching"
	@if command -v entr >/dev/null 2>&1; then \
		find $(EXTENSION_PATH) -name "*.go" | entr -c make test EXTENSION_PATH=$(EXTENSION_PATH); \
	else \
		echo "❌ entr is not installed. Install with: apt install entr (Ubuntu/Debian) or brew install entr (macOS)"; \
		exit 1; \
	fi

# Quick test all extensions in src/
.PHONY: test-all
test-all: build-tester
	@echo "🧪 Testing all extensions in src/"
	@for dir in src/*/; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			echo "Testing extension: $$dir"; \
			./$(TESTER_BINARY) -path "$$dir" || true; \
			echo ""; \
		fi; \
	done

# Development helpers
.PHONY: fmt
fmt:
	@echo "🎨 Formatting Go code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "🔍 Vetting Go code..."
	go vet ./...

.PHONY: mod-tidy
mod-tidy:
	@echo "📦 Tidying Go modules..."
	go mod tidy

# Full development cycle
.PHONY: dev-check
dev-check: fmt vet mod-tidy test-verbose
	@echo "✅ Development check complete"

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "🛠 Installing development tools..."
	@echo "Installing entr for file watching..."
	@if command -v apt >/dev/null 2>&1; then \
		sudo apt update && sudo apt install -y entr; \
	elif command -v brew >/dev/null 2>&1; then \
		brew install entr; \
	elif command -v yum >/dev/null 2>&1; then \
		sudo yum install -y entr; \
	else \
		echo "Please install 'entr' manually for file watching functionality"; \
	fi

# Create a new extension template
.PHONY: new-extension
new-extension:
	@if [ -z "$(NAME)" ]; then \
		echo "❌ Please specify extension name: make new-extension NAME=myextension"; \
		exit 1; \
	fi
	@echo "🆕 Creating new extension: $(NAME)"
	@mkdir -p src/$(NAME)
	@cp src/allanime/main.go src/$(NAME)/main.go
	@sed -i 's/allanime/$(NAME)/g' src/$(NAME)/main.go
	@sed -i 's/AllAnime/$(shell echo $(NAME) | sed 's/^./\U&/')Extension/g' src/$(NAME)/main.go
	@echo "✅ New extension created at: src/$(NAME)"
	@echo "💡 Edit src/$(NAME)/main.go to customize your extension"

# Generate test report
.PHONY: report
report: build-tester
	@echo "📊 Generating comprehensive test report..."
	@mkdir -p reports
	@echo "# Extension Test Report" > reports/test-report.md
	@echo "Generated on: $$(date)" >> reports/test-report.md
	@echo "" >> reports/test-report.md
	@for dir in src/*/; do \
		if [ -d "$$dir" ] && [ -f "$$dir/main.go" ]; then \
			echo "## Extension: $$dir" >> reports/test-report.md; \
			echo '```' >> reports/test-report.md; \
			./$(TESTER_BINARY) -path "$$dir" >> reports/test-report.md 2>&1 || true; \
			echo '```' >> reports/test-report.md; \
			echo "" >> reports/test-report.md; \
		fi; \
	done
	@echo "✅ Test report generated: reports/test-report.md"
