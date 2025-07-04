package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TestResult represents the result of a single test
type TestResult struct {
	Name        string `json:"name"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message"`
	Details     string `json:"details,omitempty"`
	Duration    string `json:"duration"`
	Suggestions string `json:"suggestions,omitempty"`
}

// ExtensionTestReport represents the complete test report for an extension
type ExtensionTestReport struct {
	ExtensionPath   string       `json:"extension_path"`
	ExtensionName   string       `json:"extension_name"`
	TestsRun        int          `json:"tests_run"`
	TestsPassed     int          `json:"tests_passed"`
	TestsFailed     int          `json:"tests_failed"`
	OverallResult   bool         `json:"overall_result"`
	Duration        string       `json:"duration"`
	BinaryPath      string       `json:"binary_path,omitempty"`
	ExtensionInfo   interface{}  `json:"extension_info,omitempty"`
	WorkingSources  []string     `json:"working_sources"`
	FailedSources   []string     `json:"failed_sources"`
	Tests           []TestResult `json:"tests"`
	Recommendations []string     `json:"recommendations"`
}

// ExtensionInfo represents the structure returned by extension-info command
type ExtensionInfo struct {
	Name    string       `json:"name"`
	Package string       `json:"pkg"`
	Lang    string       `json:"lang"`
	Version string       `json:"version"`
	NSFW    bool         `json:"nsfw"`
	Sources []SourceInfo `json:"sources"`
}

// SourceInfo represents individual source information
type SourceInfo struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	BaseURL              string `json:"baseURL"`
	Language             string `json:"language"`
	NSFW                 bool   `json:"nsfw"`
	RateLimit            int    `json:"rateLimit"`
	SupportsLatest       bool   `json:"supportsLatest"`
	SupportsSearch       bool   `json:"supportsSearch"`
	SupportsRelatedAnime bool   `json:"supportsRelatedAnime"`
}

// ExtensionTester handles testing of extensions
type ExtensionTester struct {
	extensionPath string
	binaryPath    string
	report        *ExtensionTestReport
	verbose       bool
	outputFormat  string
}

// NewExtensionTester creates a new extension tester
func NewExtensionTester(extensionPath string, verbose bool, outputFormat string) *ExtensionTester {
	return &ExtensionTester{
		extensionPath: extensionPath,
		verbose:       verbose,
		outputFormat:  outputFormat,
		report: &ExtensionTestReport{
			ExtensionPath:   extensionPath,
			Tests:           []TestResult{},
			WorkingSources:  []string{},
			FailedSources:   []string{},
			Recommendations: []string{},
		},
	}
}

// runTest executes a single test and records the result
func (et *ExtensionTester) runTest(name string, testFunc func() (bool, string, string)) {
	start := time.Now()

	if et.verbose {
		fmt.Printf("🧪 Running test: %s\n", name)
	}

	passed, message, details := testFunc()
	duration := time.Since(start)

	result := TestResult{
		Name:     name,
		Passed:   passed,
		Message:  message,
		Details:  details,
		Duration: duration.String(),
	}

	// Add suggestions based on test results
	if !passed {
		result.Suggestions = et.getSuggestions(name, message)
	}

	et.report.Tests = append(et.report.Tests, result)
	et.report.TestsRun++

	if passed {
		et.report.TestsPassed++
		if et.verbose {
			fmt.Printf("  ✅ %s\n", message)
		}
	} else {
		et.report.TestsFailed++
		if et.verbose {
			fmt.Printf("  ❌ %s\n", message)
			if details != "" {
				fmt.Printf("     Details: %s\n", details)
			}
			if result.Suggestions != "" {
				fmt.Printf("     💡 Suggestion: %s\n", result.Suggestions)
			}
		}
	}
}

// getSuggestions provides helpful suggestions based on test failures
func (et *ExtensionTester) getSuggestions(testName, message string) string {
	suggestions := map[string]string{
		"Build Extension":        "Ensure your Go code compiles without errors. Check for missing dependencies in go.mod.",
		"Extension Info Command": "Implement the GetExtensionInfo() method that returns proper ExtensionInfo structure.",
		"JSON Validation":        "Make sure your commands output valid JSON. Use json.Marshal() for consistent formatting.",
		"Source Testing":         "Verify your scraper can connect to the target website and handle rate limits properly.",
		"Search Functionality":   "Implement proper search logic that can handle common anime titles like 'naruto', 'one piece'.",
		"Episode Listing":        "Ensure your GetEpisodeList() method returns episodes with proper ID and episode numbers.",
		"Stream URL Generation":  "Implement GetVideoList() that returns working stream URLs with proper headers.",
	}

	for key, suggestion := range suggestions {
		if strings.Contains(testName, key) {
			return suggestion
		}
	}

	return "Check the implementation.md for detailed requirements and examples."
}

// buildExtension compiles the extension
func (et *ExtensionTester) buildExtension() (bool, string, string) {
	// Get extension name from path
	et.report.ExtensionName = filepath.Base(et.extensionPath)

	// Get absolute path for extension directory
	absExtensionPath, err := filepath.Abs(et.extensionPath)
	if err != nil {
		return false, "Failed to get absolute path", err.Error()
	}

	// Create binary path
	binaryName := fmt.Sprintf("%s-test", et.report.ExtensionName)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	et.binaryPath = filepath.Join(absExtensionPath, binaryName)

	// Build command
	cmd := exec.Command("go", "build", "-o", binaryName, ".")
	cmd.Dir = absExtensionPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "Failed to build extension", string(output)
	}

	// Verify binary exists
	if _, err := os.Stat(et.binaryPath); os.IsNotExist(err) {
		return false, "Binary not created", "Go build succeeded but binary file not found"
	}

	et.report.BinaryPath = et.binaryPath
	return true, "Extension built successfully", ""
}

// testExtensionInfo tests the extension-info command
func (et *ExtensionTester) testExtensionInfo() (bool, string, string) {
	output, err := et.runCommand("extension-info")
	if err != nil {
		return false, "Extension-info command failed", err.Error()
	}

	// Validate JSON
	var extInfo ExtensionInfo
	if err := json.Unmarshal([]byte(output), &extInfo); err != nil {
		return false, "Invalid JSON output", fmt.Sprintf("JSON parse error: %v", err)
	}

	// Check required fields
	missing := []string{}
	if extInfo.Name == "" {
		missing = append(missing, "name")
	}
	if extInfo.Package == "" {
		missing = append(missing, "package")
	}
	if extInfo.Version == "" {
		missing = append(missing, "version")
	}
	if len(extInfo.Sources) == 0 {
		missing = append(missing, "sources")
	}

	if len(missing) > 0 {
		return false, "Missing required fields", fmt.Sprintf("Missing: %s", strings.Join(missing, ", "))
	}

	et.report.ExtensionInfo = extInfo
	return true, fmt.Sprintf("Extension info valid (%d sources found)", len(extInfo.Sources)), ""
}

// testAllSources tests all sources in the extension
func (et *ExtensionTester) testAllSources() (bool, string, string) {
	extInfo, ok := et.report.ExtensionInfo.(ExtensionInfo)
	if !ok {
		return false, "No extension info available", "Run extension-info test first"
	}

	if len(extInfo.Sources) == 0 {
		return false, "No sources to test", "Extension has no sources defined"
	}

	totalSources := len(extInfo.Sources)
	workingSources := 0
	details := []string{}

	for _, source := range extInfo.Sources {
		if et.verbose {
			fmt.Printf("  🔍 Testing source: %s (ID: %s)\n", source.Name, source.ID)
		}

		// Test source-info
		if !et.testSourceInfo(source.ID) {
			et.report.FailedSources = append(et.report.FailedSources, source.Name)
			details = append(details, fmt.Sprintf("%s: source-info failed", source.Name))
			continue
		}

		// Test search functionality
		if !et.testSourceSearch(source.ID, source.Name) {
			et.report.FailedSources = append(et.report.FailedSources, source.Name)
			details = append(details, fmt.Sprintf("%s: search failed", source.Name))
			continue
		}

		// Test full pipeline (search → episodes → streams)
		if et.testSourcePipeline(source.ID, source.Name) {
			et.report.WorkingSources = append(et.report.WorkingSources, source.Name)
			workingSources++
			details = append(details, fmt.Sprintf("%s: ✅ full pipeline working", source.Name))
		} else {
			et.report.FailedSources = append(et.report.FailedSources, source.Name)
			details = append(details, fmt.Sprintf("%s: pipeline incomplete", source.Name))
		}
	}

	if workingSources == 0 {
		return false, "No sources working", strings.Join(details, "; ")
	}

	return true, fmt.Sprintf("%d/%d sources working", workingSources, totalSources), strings.Join(details, "; ")
}

// testSourceInfo tests source-info command for a specific source
func (et *ExtensionTester) testSourceInfo(sourceID string) bool {
	// Try with source ID first, fallback to no source ID
	output, err := et.runCommand("source-info", "--source", sourceID)
	if err != nil {
		output, err = et.runCommand("source-info")
		if err != nil {
			return false
		}
	}

	// Validate JSON
	var sourceInfo map[string]interface{}
	return json.Unmarshal([]byte(output), &sourceInfo) == nil
}

// testSourceSearch tests search functionality for a source
func (et *ExtensionTester) testSourceSearch(sourceID, sourceName string) bool {
	queries := []string{"naruto", "one piece", "attack on titan"}

	for _, query := range queries {
		// Try with source ID first, fallback to no source ID
		output, err := et.runCommand("search", "--query", query, "--page", "1", "--source", sourceID)
		if err != nil {
			output, err = et.runCommand("search", "--query", query, "--page", "1")
			if err != nil {
				continue
			}
		}

		// Check if we got results
		var results []interface{}
		if json.Unmarshal([]byte(output), &results) == nil && len(results) > 0 {
			return true
		}
	}

	return false
}

// testSourcePipeline tests the complete pipeline: search → episodes → streams
func (et *ExtensionTester) testSourcePipeline(sourceID, sourceName string) bool {
	queries := []string{"naruto", "one piece", "attack on titan"}

	for _, query := range queries {
		// Search
		searchOutput, err := et.runCommand("search", "--query", query, "--page", "1")
		if err != nil {
			continue
		}

		var searchResults []map[string]interface{}
		if json.Unmarshal([]byte(searchOutput), &searchResults) != nil || len(searchResults) == 0 {
			continue
		}

		animeID, ok := searchResults[0]["anime_id"].(string)
		if !ok {
			continue
		}

		// Episodes
		episodesOutput, err := et.runCommand("episodes", "--anime", animeID)
		if err != nil {
			continue
		}

		var episodes []map[string]interface{}
		if json.Unmarshal([]byte(episodesOutput), &episodes) != nil || len(episodes) == 0 {
			continue
		}

		episodeNumber, ok := episodes[0]["episode_number"].(float64)
		if !ok {
			continue
		}

		// Streams
		streamOutput, err := et.runCommand("stream-url", "--anime", animeID, "--episode", fmt.Sprintf("%.0f", episodeNumber))
		if err != nil {
			continue
		}

		var streamResponse map[string]interface{}
		if json.Unmarshal([]byte(streamOutput), &streamResponse) != nil {
			continue
		}

		streams, ok := streamResponse["streams"].([]interface{})
		if !ok || len(streams) == 0 {
			continue
		}

		// Test if first stream URL is accessible
		firstStream, ok := streams[0].(map[string]interface{})
		if !ok {
			continue
		}

		videoURL, ok := firstStream["videourl"].(string)
		if !ok || videoURL == "" {
			continue
		}

		// Quick accessibility test
		if et.testURLAccessibility(videoURL) {
			return true
		}
	}

	return false
}

// testURLAccessibility tests if a URL is accessible
func (et *ExtensionTester) testURLAccessibility(url string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx, 3xx, 401, 403, 405 as "accessible"
	return resp.StatusCode < 400 || resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 405
}

// testCommandStructure tests if all required commands are implemented
func (et *ExtensionTester) testCommandStructure() (bool, string, string) {
	requiredCommands := []string{"extension-info", "list-sources", "source-info", "search", "episodes", "stream-url"}
	implemented := []string{}
	missing := []string{}

	for _, cmd := range requiredCommands {
		// Test help for command
		_, err := et.runCommand(cmd, "--help")
		if err != nil {
			// Try running without help
			_, err = et.runCommand(cmd)
		}

		if err == nil {
			implemented = append(implemented, cmd)
		} else {
			missing = append(missing, cmd)
		}
	}

	if len(missing) > 0 {
		return false, fmt.Sprintf("Missing commands: %s", strings.Join(missing, ", ")),
			fmt.Sprintf("Implemented: %s", strings.Join(implemented, ", "))
	}

	return true, "All required commands implemented", ""
}

// runCommand executes a command on the built binary
func (et *ExtensionTester) runCommand(args ...string) (string, error) {
	cmd := exec.Command(et.binaryPath, args...)

	// Get absolute path for extension directory
	absExtensionPath, err := filepath.Abs(et.extensionPath)
	if err != nil {
		return "", err
	}
	cmd.Dir = absExtensionPath

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// generateRecommendations generates recommendations based on test results
func (et *ExtensionTester) generateRecommendations() {
	recommendations := []string{}

	// Check overall results
	if et.report.TestsPassed == 0 {
		recommendations = append(recommendations, "Start with implementing basic extension-info command")
	}

	if len(et.report.WorkingSources) == 0 && len(et.report.FailedSources) > 0 {
		recommendations = append(recommendations, "Focus on fixing source connectivity and search functionality")
	}

	if et.report.TestsPassed < et.report.TestsRun/2 {
		recommendations = append(recommendations, "Review implementation.md for detailed requirements")
	}

	// Specific recommendations based on failed tests
	for _, test := range et.report.Tests {
		if !test.Passed {
			switch {
			case strings.Contains(test.Name, "Build"):
				recommendations = append(recommendations, "Fix compilation errors before proceeding")
			case strings.Contains(test.Name, "JSON"):
				recommendations = append(recommendations, "Ensure all outputs are valid JSON format")
			case strings.Contains(test.Name, "Source"):
				recommendations = append(recommendations, "Check network connectivity and API endpoints")
			}
		}
	}

	et.report.Recommendations = recommendations
}

// printReport prints the test report in the specified format
func (et *ExtensionTester) printReport() {
	et.report.OverallResult = et.report.TestsPassed > 0 && len(et.report.WorkingSources) > 0

	switch et.outputFormat {
	case "json":
		et.printJSONReport()
	case "detailed":
		et.printDetailedReport()
	default:
		et.printSummaryReport()
	}
}

// printSummaryReport prints a concise summary report
func (et *ExtensionTester) printSummaryReport() {
	fmt.Printf("\n🎯 Extension Test Summary\n")
	fmt.Printf("========================\n")
	fmt.Printf("Extension: %s\n", et.report.ExtensionName)
	fmt.Printf("Path: %s\n", et.report.ExtensionPath)
	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("  Tests Run: %d\n", et.report.TestsRun)
	fmt.Printf("  Passed: %d ✅\n", et.report.TestsPassed)
	fmt.Printf("  Failed: %d ❌\n", et.report.TestsFailed)
	fmt.Printf("  Success Rate: %.1f%%\n", float64(et.report.TestsPassed)/float64(et.report.TestsRun)*100)

	if len(et.report.WorkingSources) > 0 {
		fmt.Printf("\n🔄 Working Sources (%d):\n", len(et.report.WorkingSources))
		for _, source := range et.report.WorkingSources {
			fmt.Printf("  ✅ %s\n", source)
		}
	}

	if len(et.report.FailedSources) > 0 {
		fmt.Printf("\n❌ Failed Sources (%d):\n", len(et.report.FailedSources))
		for _, source := range et.report.FailedSources {
			fmt.Printf("  ❌ %s\n", source)
		}
	}

	if len(et.report.Recommendations) > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		for i, rec := range et.report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	fmt.Printf("\n🏆 Overall Result: ")
	if et.report.OverallResult {
		fmt.Printf("✅ PASS - Extension is working!\n")
	} else {
		fmt.Printf("❌ FAIL - Extension needs fixes\n")
	}
}

// printDetailedReport prints a detailed report with all test results
func (et *ExtensionTester) printDetailedReport() {
	et.printSummaryReport()

	fmt.Printf("\n📋 Detailed Test Results:\n")
	fmt.Printf("==========================\n")
	for i, test := range et.report.Tests {
		status := "❌"
		if test.Passed {
			status = "✅"
		}

		fmt.Printf("%d. %s %s\n", i+1, status, test.Name)
		fmt.Printf("   Message: %s\n", test.Message)
		fmt.Printf("   Duration: %s\n", test.Duration)

		if test.Details != "" {
			fmt.Printf("   Details: %s\n", test.Details)
		}

		if test.Suggestions != "" {
			fmt.Printf("   💡 Suggestion: %s\n", test.Suggestions)
		}
		fmt.Printf("\n")
	}
}

// printJSONReport prints the report in JSON format
func (et *ExtensionTester) printJSONReport() {
	jsonData, err := json.MarshalIndent(et.report, "", "  ")
	if err != nil {
		fmt.Printf("Error generating JSON report: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// RunTests executes all tests for the extension
func (et *ExtensionTester) RunTests() {
	start := time.Now()

	fmt.Printf("🚀 Starting extension tests for: %s\n", et.extensionPath)

	// Test 1: Build Extension
	et.runTest("Build Extension", et.buildExtension)
	if et.report.TestsFailed > 0 {
		fmt.Printf("❌ Build failed, stopping tests\n")
		et.report.Duration = time.Since(start).String()
		et.generateRecommendations()
		et.printReport()
		return
	}

	// Test 2: Extension Info
	et.runTest("Extension Info Command", et.testExtensionInfo)

	// Test 3: Command Structure
	et.runTest("Command Structure", et.testCommandStructure)

	// Test 4: Source Testing
	et.runTest("Source Testing", et.testAllSources)

	et.report.Duration = time.Since(start).String()
	et.generateRecommendations()

	// Cleanup binary
	if et.binaryPath != "" {
		os.Remove(et.binaryPath)
	}

	et.printReport()
}

func main() {
	var (
		extensionPath = flag.String("path", ".", "Path to the extension directory")
		verbose       = flag.Bool("verbose", false, "Enable verbose output")
		outputFormat  = flag.String("format", "summary", "Output format: summary, detailed, json")
		help          = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Extension Tester - Test your anime extensions locally\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Test extension in current directory\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Test specific extension with verbose output\n")
		fmt.Fprintf(os.Stderr, "  %s -path ./src/allanime -verbose\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Get detailed report in JSON format\n")
		fmt.Fprintf(os.Stderr, "  %s -path ./src/myextension -format json\n\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Validate extension path
	if _, err := os.Stat(*extensionPath); os.IsNotExist(err) {
		fmt.Printf("❌ Extension path does not exist: %s\n", *extensionPath)
		os.Exit(1)
	}

	// Check for go.mod or .go files
	hasGoFiles := false
	entries, err := os.ReadDir(*extensionPath)
	if err != nil {
		fmt.Printf("❌ Cannot read extension directory: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") || entry.Name() == "go.mod" {
			hasGoFiles = true
			break
		}
	}

	if !hasGoFiles {
		fmt.Printf("❌ No Go files found in extension directory: %s\n", *extensionPath)
		os.Exit(1)
	}

	// Validate output format
	validFormats := map[string]bool{"summary": true, "detailed": true, "json": true}
	if !validFormats[*outputFormat] {
		fmt.Printf("❌ Invalid output format: %s (valid: summary, detailed, json)\n", *outputFormat)
		os.Exit(1)
	}

	// Create and run tester
	tester := NewExtensionTester(*extensionPath, *verbose, *outputFormat)
	tester.RunTests()

	// Exit with appropriate code
	if tester.report.OverallResult {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
