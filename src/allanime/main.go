package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/wraient/pair/pkg/scraper"
)

type AllanimeScaper struct {
	agent        string
	allanimeRef  string
	allanimeBase string
	allanimeAPI  string
}

// decodeProviderID decodes the encoded provider ID to get the actual URL
func (s *AllanimeScaper) decodeProviderID(encoded string) string {
	re := regexp.MustCompile("..")
	pairs := re.FindAllString(encoded, -1)

	replacements := map[string]string{
		"01": "9", "08": "0", "05": "=", "0a": "2", "0b": "3", "0c": "4", "07": "?",
		"00": "8", "5c": "d", "0f": "7", "5e": "f", "17": "/", "54": "l", "09": "1",
		"48": "p", "4f": "w", "0e": "6", "5b": "c", "5d": "e", "0d": "5", "53": "k",
		"1e": "&", "5a": "b", "59": "a", "4a": "r", "4c": "t", "4e": "v", "57": "o",
		"51": "i",
	}

	for i, pair := range pairs {
		if val, exists := replacements[pair]; exists {
			pairs[i] = val
		}
	}

	result := strings.Join(pairs, "")
	result = strings.ReplaceAll(result, "/clock", "/clock.json")
	return result
}

// extractLinks retrieves the actual stream links from the provider
func (s *AllanimeScaper) extractLinks(provider_id string) map[string]interface{} {
	url := "https://" + s.allanimeBase + provider_id
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	var videoData map[string]interface{}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return nil
	}

	req.Header.Set("Referer", s.allanimeRef)
	req.Header.Set("User-Agent", s.agent)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		return nil
	}

	err = json.Unmarshal(body, &videoData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return nil
	}

	return videoData
}

// Anime represents the structure of anime data from the API
type anime struct {
	ID                string      `json:"_id"`
	Name              string      `json:"name"`
	EnglishName       string      `json:"englishName"`
	AvailableEpisodes interface{} `json:"availableEpisodes"`
}

// Response represents the structure of the API response
type response struct {
	Data struct {
		Shows struct {
			Edges []anime `json:"edges"`
		} `json:"shows"`
	} `json:"data"`
}

func NewAllanimeScaper() *AllanimeScaper {
	const (
		agent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"
		allanimeRef  = "https://allanime.to"
		allanimeBase = "allanime.day"
		allanimeAPI  = "https://api." + allanimeBase + "/api"
	)

	return &AllanimeScaper{
		agent:        agent,
		allanimeRef:  allanimeRef,
		allanimeBase: allanimeBase,
		allanimeAPI:  allanimeAPI,
	}
}

// GetInfo returns metadata about this scraper implementation
func (s *AllanimeScaper) GetInfo() scraper.ScraperInfo {
	return scraper.ScraperInfo{
		ID:          "allanime",
		Name:        "AllAnime",
		Version:     "0.1.0",
		Description: "Scraper for AllAnime - a popular anime streaming website",
	}
}

// Search searches for anime matching the given query
func (s *AllanimeScaper) Search(query string, mode string) ([]scraper.SearchResult, error) {
	searchGql := `query($search: SearchInput, $limit: Int, $page: Int, $translationType: VaildTranslationTypeEnumType, $countryOrigin: VaildCountryOriginEnumType) {
		shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin) {
			edges {
				_id
				name
				englishName
				availableEpisodes
			}
		}
	}`

	// Prepare the GraphQL variables
	variables := map[string]interface{}{
		"search": map[string]interface{}{
			"allowAdult":   false,
			"allowUnknown": false,
			"query":        query,
		},
		"limit":           40,
		"page":            1,
		"translationType": mode,
		"countryOrigin":   "ALL",
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error encoding variables: %v", err)
	}

	// Build the request URL
	reqURL := fmt.Sprintf("%s?variables=%s&query=%s", s.allanimeAPI, url.QueryEscape(string(variablesJSON)), url.QueryEscape(searchGql))

	// Make the HTTP request
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", s.agent)
	req.Header.Set("Referer", s.allanimeRef)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var apiResponse response
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	searchResults := make([]scraper.SearchResult, 0)
	for _, anime := range apiResponse.Data.Shows.Edges {
		// Future: Could include total episodes count in the result if needed
		alternateTitles := make(map[string]string)
		if anime.EnglishName != "" {
			alternateTitles["en"] = anime.EnglishName
		}

		searchResults = append(searchResults, scraper.SearchResult{
			ID:              anime.ID,
			Title:           anime.Name,
			AlternateTitles: alternateTitles,
			Type:            "TV",
			Status:          "Unknown",
			Thumbnail:       "", // We'd need to add this to the GraphQL query
		})
	}

	return searchResults, nil
}

// GetEpisodeList retrieves the list of available episodes for an anime
func (s *AllanimeScaper) GetEpisodeList(animeID string, mode string) ([]scraper.EpisodeInfo, error) {
	episodesListGql := `query ($showId: String!) { show( _id: $showId ) { _id availableEpisodesDetail }}`

	variables := map[string]interface{}{
		"showId": animeID,
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error encoding variables: %v", err)
	}

	reqURL := fmt.Sprintf("%s?variables=%s&query=%s", s.allanimeAPI, url.QueryEscape(string(variablesJSON)), url.QueryEscape(episodesListGql))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", s.agent)
	req.Header.Set("Referer", s.allanimeRef)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var response struct {
		Data struct {
			Show struct {
				ID                      string                 `json:"_id"`
				AvailableEpisodesDetail map[string]interface{} `json:"availableEpisodesDetail"`
			} `json:"show"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	var episodeNumbers []float64
	if eps, ok := response.Data.Show.AvailableEpisodesDetail[mode].([]interface{}); ok {
		for _, ep := range eps {
			if epNum, err := strconv.ParseFloat(fmt.Sprintf("%v", ep), 64); err == nil {
				episodeNumbers = append(episodeNumbers, epNum)
			}
		}
	}

	// Sort episodes numerically
	sort.Float64s(episodeNumbers)

	episodeList := make([]scraper.EpisodeInfo, len(episodeNumbers))
	for i, epNum := range episodeNumbers {
		episodeList[i] = scraper.EpisodeInfo{
			Number: epNum,
		}
	}

	return episodeList, nil
}

// GetStreamInfo retrieves stream information for a specific episode
func (s *AllanimeScaper) GetStreamInfo(animeID string, episodeNumber float64, mode string) ([]scraper.StreamInfo, error) {
	query := `query($showId:String!,$translationType:VaildTranslationTypeEnumType!,$episodeString:String!){episode(showId:$showId,translationType:$translationType,episodeString:$episodeString){episodeString sourceUrls}}`

	variables := map[string]interface{}{
		"showId":          animeID,
		"translationType": mode,
		"episodeString":   fmt.Sprintf("%v", episodeNumber),
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error encoding variables: %v", err)
	}

	reqURL := fmt.Sprintf("%s?variables=%s&query=%s", s.allanimeAPI, url.QueryEscape(string(variablesJSON)), url.QueryEscape(query))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", s.agent)
	req.Header.Set("Referer", s.allanimeRef)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var response struct {
		Data struct {
			Episode struct {
				SourceUrls []struct {
					SourceUrl string `json:"sourceUrl"`
				} `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	validURLs := make([]string, 0)
	highestPriority := -1
	var highestPriorityURL string

	// LinkPriorities lists the preferred domains in order
	LinkPriorities := []string{"kraken", "dolphin", "duck"}

	// First pass: collect valid URLs and find highest priority one
	for _, url := range response.Data.Episode.SourceUrls {
		if len(url.SourceUrl) > 2 && unicode.IsDigit(rune(url.SourceUrl[2])) {
			if strings.HasPrefix(url.SourceUrl, "--") {
				decodedURL := s.decodeProviderID(url.SourceUrl[2:])
				if strings.Contains(decodedURL, LinkPriorities[0]) {
					priority := int(url.SourceUrl[2] - '0')
					if priority > highestPriority {
						highestPriority = priority
						highestPriorityURL = url.SourceUrl
					}
				} else {
					validURLs = append(validURLs, url.SourceUrl)
				}
			} else {
				validURLs = append(validURLs, url.SourceUrl)
			}
		}
	}

	// If we found a highest priority URL, use only that
	if highestPriorityURL != "" {
		validURLs = []string{highestPriorityURL}
	}

	if len(validURLs) == 0 {
		return nil, fmt.Errorf("no valid source URLs found in response")
	}

	// Create channels for results and a slice to store ordered results
	results := make(chan struct {
		index int
		links []scraper.StreamInfo
		err   error
	}, len(validURLs))
	orderedResults := make([][]scraper.StreamInfo, len(validURLs))

	// Add a channel for high priority links
	highPriorityLink := make(chan []scraper.StreamInfo, 1)

	// Create rate limiter
	rateLimiter := time.NewTicker(50 * time.Millisecond)
	defer rateLimiter.Stop()

	// Launch goroutines
	// remainingURLs := len(validURLs)
	for i, sourceUrl := range validURLs {
		go func(idx int, url string) {
			<-rateLimiter.C // Rate limit the requests

			decodedProviderID := s.decodeProviderID(url[2:])
			extractedLinks := s.extractLinks(decodedProviderID)

			if extractedLinks == nil {
				results <- struct {
					index int
					links []scraper.StreamInfo
					err   error
				}{
					index: idx,
					err:   fmt.Errorf("failed to extract links for provider %s", decodedProviderID),
				}
				return
			}

			linksInterface, ok := extractedLinks["links"].([]interface{})
			if !ok {
				results <- struct {
					index int
					links []scraper.StreamInfo
					err   error
				}{
					index: idx,
					err:   fmt.Errorf("links field is not []interface{} for provider %s", decodedProviderID),
				}
				return
			}

			var streamInfos []scraper.StreamInfo
			for _, linkInterface := range linksInterface {
				linkMap, ok := linkInterface.(map[string]interface{})
				if !ok {
					continue
				}

				link, ok := linkMap["link"].(string)
				if !ok {
					continue
				}

				quality, _ := linkMap["resolutionStr"].(string)
				// Only decode if the link starts with "--", otherwise use as is
				var finalURL string
				if strings.HasPrefix(link, "--") {
					decodedLink := s.decodeProviderID(link[2:])
					// Ensure the URL starts with https:// if it doesn't already
					if !strings.HasPrefix(decodedLink, "http") {
						finalURL = "https://" + decodedLink
					} else {
						finalURL = decodedLink
					}
				} else {
					finalURL = link
				}

				streamInfo := scraper.StreamInfo{
					URL:     finalURL,
					Quality: quality,
					Format:  "mp4", // Default format
				}
				streamInfos = append(streamInfos, streamInfo)

				// Check if this is a high priority link
				for _, domain := range LinkPriorities[:3] { // Check only top 3 priority domains
					if strings.Contains(link, domain) {
						select {
						case highPriorityLink <- []scraper.StreamInfo{streamInfo}:
						default:
							// Channel already has a high priority link
						}
						break
					}
				}
			}

			results <- struct {
				index int
				links []scraper.StreamInfo
				err   error
			}{
				index: idx,
				links: streamInfos,
			}
		}(i, sourceUrl)
	}

	// Collect results with timeout
	timeout := time.After(10 * time.Second)
	var collectedErrors []error
	successCount := 0

	// First, try to get a high priority link
	select {
	case links := <-highPriorityLink:
		return links, nil
	case <-time.After(2 * time.Second): // Wait only briefly for high priority link
		// No high priority link found quickly, proceed with normal collection
	}

	// Collect results maintaining order
	for successCount < len(validURLs) {
		select {
		case res := <-results:
			if res.err != nil {
				collectedErrors = append(collectedErrors, res.err)
			} else {
				orderedResults[res.index] = res.links
				successCount++
			}
		case <-timeout:
			if successCount > 0 {
				// Flatten available results
				var allStreams []scraper.StreamInfo
				for _, streams := range orderedResults {
					allStreams = append(allStreams, streams...)
				}
				return allStreams, nil
			}
			return nil, fmt.Errorf("timeout waiting for results")
		}
	}

	// Flatten and return results
	var allStreams []scraper.StreamInfo
	for _, streams := range orderedResults {
		allStreams = append(allStreams, streams...)
	}
	if len(allStreams) == 0 {
		return nil, fmt.Errorf("no valid links found from %d URLs: %v", len(validURLs), collectedErrors)
	}

	return allStreams, nil
}

func main() {
	action := flag.String("action", "info", "Action to perform: info, search, episodes, streams")
	query := flag.String("query", "", "Search query (for action=search)")
	id := flag.String("id", "", "Anime ID for episodes/streams")
	episode := flag.Float64("episode", 0, "Episode number for streams")
	mode := flag.String("mode", "sub", "Mode: sub or dub")

	flag.Parse()

	scraper := NewAllanimeScaper()

	var result interface{}
	var err error

	switch *action {
	case "info":
		result = scraper.GetInfo()
	case "search":
		if *query == "" {
			err = fmt.Errorf("search query is required for action 'search'")
			break
		}
		result, err = scraper.Search(*query, *mode)
	case "episodes":
		if *id == "" {
			err = fmt.Errorf("anime ID is required for action 'episodes'")
			break
		}
		result, err = scraper.GetEpisodeList(*id, *mode)
	case "streams":
		if *id == "" || *episode == 0 {
			err = fmt.Errorf("anime ID and episode number are required for action 'streams'")
			break
		}
		result, err = scraper.GetStreamInfo(*id, *episode, *mode)
	default:
		err = fmt.Errorf("invalid action: %s", *action)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output the result as JSON
	jsonOutput, jsonErr := json.MarshalIndent(result, "", "  ")
	if jsonErr != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling result to JSON: %v\n", jsonErr)
		os.Exit(1)
	}
	fmt.Println(string(jsonOutput))
}
