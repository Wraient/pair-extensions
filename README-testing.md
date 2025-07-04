# Extension Tester

A comprehensive testing tool for anime extension developers to validate their extensions locally before submitting them.

## Features

- 🔨 **Build Testing**: Automatically builds your extension from source
- 📋 **Command Validation**: Tests all required commands (extension-info, search, episodes, etc.)
- 🔍 **Source Testing**: Tests all sources defined in your extension individually
- 🌐 **End-to-End Testing**: Validates the complete pipeline: search → episodes → streams
- 📊 **Comprehensive Reporting**: Detailed reports with suggestions for fixes
- 💡 **Smart Recommendations**: Provides actionable advice based on test failures

## Installation

### Option 1: Build from Source
```bash
go build -o extension-tester test-extension.go
```

### Option 2: Run Directly
```bash
go run test-extension.go [OPTIONS]
```

## Usage

### Basic Testing
Test extension in current directory:
```bash
./extension-tester
```

### Test Specific Extension
```bash
./extension-tester -path ./src/allanime
```

### Verbose Output
See detailed test execution:
```bash
./extension-tester -path ./src/myextension -verbose
```

### Different Output Formats

**Summary Report (default)**:
```bash
./extension-tester -format summary
```

**Detailed Report**:
```bash
./extension-tester -format detailed
```

**JSON Report** (for automated tools):
```bash
./extension-tester -format json
```

## Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-path` | `.` | Path to the extension directory |
| `-verbose` | `false` | Enable verbose output during testing |
| `-format` | `summary` | Output format: `summary`, `detailed`, or `json` |
| `-help` | `false` | Show help message |

## Example Output

### Summary Report
```
🎯 Extension Test Summary
========================
Extension: allanime
Path: ./src/allanime

📊 Results:
  Tests Run: 4
  Passed: 4 ✅
  Failed: 0 ❌
  Success Rate: 100.0%

🔄 Working Sources (1):
  ✅ AllAnime

🏆 Overall Result: ✅ PASS - Extension is working!
```

### JSON Report
```json
{
  "extension_path": "./src/allanime",
  "extension_name": "allanime",
  "tests_run": 4,
  "tests_passed": 4,
  "tests_failed": 0,
  "overall_result": true,
  "working_sources": ["AllAnime"],
  "failed_sources": [],
  "tests": [
    {
      "name": "Build Extension",
      "passed": true,
      "message": "Extension built successfully",
      "duration": "1.234s"
    }
  ],
  "recommendations": []
}
```

## What Gets Tested

### 1. Build Process
- ✅ Compiles your Go code
- ✅ Creates executable binary
- ✅ Verifies binary is runnable

### 2. Command Structure
- ✅ `extension-info` - Extension metadata
- ✅ `list-sources` - Available sources
- ✅ `source-info` - Source details
- ✅ `search` - Search functionality
- ✅ `episodes` - Episode listing
- ✅ `stream-url` - Video stream URLs

### 3. JSON Validation
- ✅ All outputs are valid JSON
- ✅ Required fields are present
- ✅ Data types are correct

### 4. Source Testing
- ✅ Each source is tested individually
- ✅ Search with common anime titles
- ✅ Episode retrieval
- ✅ Stream URL generation
- ✅ URL accessibility checks

### 5. Implementation Compliance
- ✅ Follows the specification in `implementation.md`
- ✅ Proper error handling
- ✅ Consistent data structures

## Exit Codes

- `0`: All tests passed, extension is working
- `1`: Some tests failed, extension needs fixes

## Troubleshooting

### Common Issues

**Build Failures**:
- Ensure `go.mod` is present and correct
- Check for compilation errors in your code
- Verify all dependencies are available

**JSON Validation Errors**:
- Use `json.Marshal()` for consistent JSON output
- Ensure all required fields are present
- Check for proper struct tags

**Source Test Failures**:
- Verify network connectivity
- Check API endpoints and rate limits
- Ensure proper error handling for network requests

**No Search Results**:
- Test with common anime titles manually
- Check search query formatting
- Verify the target website is accessible

### Getting Help

1. Run with `-verbose` flag for detailed output
2. Check the suggestions in the test report
3. Review `implementation.md` for requirements
4. Use `-format detailed` for comprehensive test breakdown

## Integration with Development Workflow

### Pre-commit Testing
Add to your development workflow:
```bash
# Test before committing
./extension-tester -path ./src/myextension
if [ $? -eq 0 ]; then
    git commit -m "Extension tests passing"
else
    echo "Fix extension tests before committing"
    exit 1
fi
```

### CI/CD Integration
Use JSON output for automated testing:
```bash
./extension-tester -format json > test-results.json
```

### Continuous Development
Use in watch mode during development:
```bash
# Using entr (install with: apt install entr)
find ./src/myextension -name "*.go" | entr -c ./extension-tester -path ./src/myextension -verbose
```

## Architecture

The tester follows the same validation logic as the GitHub Actions workflow:

1. **Discovery**: Finds and validates the extension directory
2. **Build**: Compiles the extension binary
3. **Validation**: Tests command structure and JSON output
4. **Source Testing**: Individual source validation
5. **Pipeline Testing**: End-to-end functionality
6. **Reporting**: Comprehensive results with recommendations

This ensures that extensions passing local tests will also pass CI/CD validation.
