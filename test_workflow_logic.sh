#!/bin/bash

# Test script to simulate the workflow logic locally
set -e

echo "=== Testing Workflow Logic Locally ==="
echo ""

# Test extension
EXTENSION="allanime"
BINARY_PATH="./bin/allanime-linux-x86_64"

echo "Testing extension: $EXTENSION"

# Test binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found: $BINARY_PATH"
    exit 1
fi
echo "✅ Binary found: $BINARY_PATH"

# Test extension-info command
echo ""
echo "=== Testing extension-info command ==="
output=$($BINARY_PATH extension-info)
echo "Extension info output:"
echo "$output"

# Validate JSON structure
echo "$output" | jq empty || {
    echo "❌ Invalid JSON output for extension-info"
    exit 1
}

# Check required fields
name=$(echo "$output" | jq -r '.name // empty')
package=$(echo "$output" | jq -r '.pkg // empty')
version=$(echo "$output" | jq -r '.version // empty')
sources=$(echo "$output" | jq -r '.sources // empty')

if [ -z "$name" ] || [ -z "$package" ] || [ -z "$version" ] || [ -z "$sources" ]; then
    echo "❌ Missing required fields in extension-info"
    echo "Name: $name, Package: $package, Version: $version, Sources: $sources"
    exit 1
fi

echo "✅ Extension info validation passed"
echo "Name: $name"
echo "Package: $package"
echo "Version: $version"

# Test EOF approach for storing output
echo ""
echo "=== Testing EOF output storage ==="
temp_file=$(mktemp)
cat > "$temp_file" << EOF
extension_info<<DELIMITER
$output
DELIMITER
EOF

echo "Stored output to temp file:"
cat "$temp_file"

# Test reading back the JSON
stored_json=$(sed -n '/extension_info<<DELIMITER/,/DELIMITER/{/extension_info<<DELIMITER/d;/DELIMITER/d;p;}' "$temp_file")
echo ""
echo "Retrieved JSON:"
echo "$stored_json"

# Validate retrieved JSON
echo "$stored_json" | jq empty || {
    echo "❌ Retrieved JSON is invalid"
    exit 1
}

echo "✅ EOF approach works correctly"

# Test sources extraction
echo ""
echo "=== Testing sources extraction ==="
sources_json=$(echo "$stored_json" | jq -r '.sources')
sources_count=$(echo "$sources_json" | jq 'length')

echo "Found $sources_count source(s)"

for i in $(seq 0 $((sources_count-1))); do
    source=$(echo "$sources_json" | jq -r ".[$i]")
    source_id=$(echo "$source" | jq -r '.id')
    source_name=$(echo "$source" | jq -r '.name')
    
    echo "Source $((i+1)): $source_name (ID: $source_id)"
done

echo "✅ Sources extraction works correctly"

# Cleanup
rm -f "$temp_file"

echo ""
echo "=== All tests passed! ==="
