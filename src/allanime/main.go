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

	"github.com/wraient/pair/pkg/scraper"
)

type AllanimeScaper struct {
	agent        string
	allanimeRef  string
	allanimeBase string
	allanimeAPI  string
}

// NewAllanimeScaper creates a new instance of the allanime scraper
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

// GetExtensionInfo returns metadata about this scraper implementation
func (s *AllanimeScaper) GetExtensionInfo() (scraper.ExtensionInfo, error) {
	return scraper.ExtensionInfo{
		Name:    "AllAnime",
		Package: "allanime",
		Lang:    "en",
		Version: "0.1.0",
		NSFW:    false,
		Sources: []scraper.SourceInfo{
			{
				ID:                   "3160569130087668532",
				Name:                 "AllAnime",
				BaseURL:              "https://allanime.to",
				Language:             "en",
				NSFW:                 false,
				RateLimit:            50,
				SupportsLatest:       false,
				SupportsSearch:       true,
				SupportsRelatedAnime: false,
			},
		},
	}, nil
}

// GetSourceInfo retrieves metadata about a specific source
func (s *AllanimeScaper) GetSourceInfo() (scraper.SourceInfo, error) {
	return scraper.SourceInfo{
		ID:                   "3160569130087668532",
		Name:                 "AllAnime",
		BaseURL:              "https://allanime.to",
		Language:             "en",
		NSFW:                 false,
		RateLimit:            50,
		SupportsLatest:       false,
		SupportsSearch:       true,
		SupportsRelatedAnime: false,
	}, nil
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
func (s *AllanimeScaper) extractLinks(provider_id string) (map[string]interface{}, error) {
	url := "https://" + s.allanimeBase + provider_id
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Referer", s.allanimeRef)
	req.Header.Set("User-Agent", s.agent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var videoData map[string]interface{}
	err = json.Unmarshal(body, &videoData)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return videoData, nil
}

// SearchAnime searches for anime with the given query and filters
func (s *AllanimeScaper) SearchAnime(query string, page int, filters string) ([]scraper.Anime, error) {
	searchGql := `query($search: SearchInput, $limit: Int, $page: Int, $translationType: VaildTranslationTypeEnumType, $countryOrigin: VaildCountryOriginEnumType) {
		shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin) {
			edges {
				_id
				name
				englishName
				availableEpisodes
				status
				type
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
		"page":            page,
		"translationType": "sub", // Default to sub
		"countryOrigin":   "ALL",
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error encoding variables: %v", err)
	}

	reqURL := fmt.Sprintf("%s?variables=%s&query=%s", s.allanimeAPI, url.QueryEscape(string(variablesJSON)), url.QueryEscape(searchGql))

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

	// Debug: Uncomment the line below for API debugging
	// fmt.Fprintf(os.Stderr, "API Response: %s\n", string(body))

	var response struct {
		Data struct {
			Shows struct {
				Edges []struct {
					ID                string      `json:"_id"`
					Name              string      `json:"name"`
					EnglishName       string      `json:"englishName"`
					AvailableEpisodes interface{} `json:"availableEpisodes"`
					Status            string      `json:"status"`
					Type              string      `json:"type"`
				} `json:"edges"`
			} `json:"shows"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	var animes []scraper.Anime
	for _, show := range response.Data.Shows.Edges {
		var episodes int
		if eps, ok := show.AvailableEpisodes.(map[string]interface{}); ok {
			if subEps, ok := eps["sub"].(float64); ok {
				episodes = int(subEps)
			}
		}

		alternativeTitles := []string{}
		if show.EnglishName != "" {
			alternativeTitles = append(alternativeTitles, show.EnglishName)
		}

		animes = append(animes, scraper.Anime{
			ID:                show.ID,
			Title:             show.Name,
			AlternativeTitles: alternativeTitles,
			Status:            show.Status,
			Episodes:          episodes,
			SubDub:            "sub",
		})
	}

	return animes, nil
}

// GetEpisodeList retrieves the list of episodes for an anime
func (s *AllanimeScaper) GetEpisodeList(animeID string) ([]scraper.Episode, error) {
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

	var episodes []scraper.Episode
	if eps, ok := response.Data.Show.AvailableEpisodesDetail["sub"].([]interface{}); ok {
		for _, ep := range eps {
			if epNum, err := strconv.ParseFloat(fmt.Sprintf("%v", ep), 64); err == nil {
				episodes = append(episodes, scraper.Episode{
					ID:            animeID,
					EpisodeNumber: epNum,
					DateUpload:    time.Now().Unix(), // We don't have actual upload dates
				})
			}
		}
	}

	return episodes, nil
}

// LinkPriorities defines the priority order for video sources
var LinkPriorities = []string{
	"sharepoint.com",
	"wixmp.com",
	"dropbox.com",
	"wetransfer.com",
	"gogoanime.com",
}

func (s *AllanimeScaper) GetVideoList(animeID string, episodeNumber float64) (scraper.VideoResponse, error) {
	query := `query($showId:String!,$translationType:VaildTranslationTypeEnumType!,$episodeString:String!){episode(showId:$showId,translationType:$translationType,episodeString:$episodeString){episodeString sourceUrls}}`

	variables := map[string]interface{}{
		"showId":          animeID,
		"translationType": "sub",
		"episodeString":   fmt.Sprintf("%v", episodeNumber),
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return scraper.VideoResponse{}, fmt.Errorf("error encoding variables: %v", err)
	}

	values := url.Values{}
	values.Set("query", query)
	values.Set("variables", string(variablesJSON))

	reqURL := fmt.Sprintf("%s?%s", s.allanimeAPI, values.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return scraper.VideoResponse{}, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", s.agent)
	req.Header.Set("Referer", s.allanimeRef)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return scraper.VideoResponse{}, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return scraper.VideoResponse{}, fmt.Errorf("error reading response: %v", err)
	}

	var response struct {
		Data struct {
			Episode struct {
				SourceUrls []struct {
					SourceUrl  string  `json:"sourceUrl"`
					Priority   float64 `json:"priority"`
					SourceName string  `json:"sourceName"`
					Type       string  `json:"type"`
				} `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return scraper.VideoResponse{}, fmt.Errorf("error parsing response: %v", err)
	}

	type streamInfo struct {
		url      string
		quality  string
		priority int
	}

	var streams []streamInfo

	// Process all sources
	for _, source := range response.Data.Episode.SourceUrls {
		if strings.HasPrefix(source.SourceUrl, "--") {
			decodedProviderID := s.decodeProviderID(source.SourceUrl[2:])
			extractedLinks, err := s.extractLinks(decodedProviderID)
			if err != nil {
				continue
			}

			if linksInterface, ok := extractedLinks["links"].([]interface{}); ok {
				for _, linkInterface := range linksInterface {
					if linkMap, ok := linkInterface.(map[string]interface{}); ok {
						if link, ok := linkMap["link"].(string); ok {
							quality, _ := linkMap["resolutionStr"].(string)
							var finalURL string
							if strings.HasPrefix(link, "--") {
								decodedLink := s.decodeProviderID(link[2:])
								if !strings.HasPrefix(decodedLink, "http") {
									finalURL = "https://" + decodedLink
								} else {
									finalURL = decodedLink
								}
							} else {
								finalURL = link
							}

							// Check priority based on domain
							priority := -1
							for i, domain := range LinkPriorities {
								if strings.Contains(finalURL, domain) {
									priority = len(LinkPriorities) - i
									break
								}
							}

							streams = append(streams, streamInfo{
								url:      finalURL,
								quality:  quality,
								priority: priority,
							})
						}
					}
				}
			}
		} else if strings.HasPrefix(source.SourceUrl, "https://") {
			// Check priority based on domain
			priority := -1
			for i, domain := range LinkPriorities {
				if strings.Contains(source.SourceUrl, domain) {
					priority = len(LinkPriorities) - i
					break
				}
			}

			streams = append(streams, streamInfo{
				url:      source.SourceUrl,
				quality:  source.SourceName,
				priority: priority,
			})
		}
	}

	// Sort streams by priority (highest first)
	sort.Slice(streams, func(i, j int) bool {
		return streams[i].priority > streams[j].priority
	})

	// Convert to scraper.Video format
	var result []scraper.Video
	for _, stream := range streams {
		result = append(result, scraper.Video{
			ID:       animeID,
			Quality:  stream.quality,
			VideoURL: stream.url,
		})
	}

	if len(result) == 0 {
		return scraper.VideoResponse{}, fmt.Errorf("no valid streams found")
	}

	return scraper.VideoResponse{
		Streams: result,
	}, nil
}

func main() {
	// Define command-line flags
	var (
		help     = flag.Bool("h", false, "Show help message")
		query    = flag.String("query", "", "Search query")
		page     = flag.Int("page", 1, "Page number")
		filters  = flag.String("filters", "", "JSON filters")
		animeURL = flag.String("anime", "", "Anime URL")
		episode  = flag.Float64("episode", 0, "Episode number")
		sourceID = flag.String("source", "3160569130087668532", "Source ID (optional, defaults to allanime)")
	)

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] COMMAND [ARGS]...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for interacting with anime video sources.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  episodes        Get the list of episodes for an anime.\n")
		fmt.Fprintf(os.Stderr, "  extension-info  Get information about a specific extension.\n")
		fmt.Fprintf(os.Stderr, "  list-sources    List all available anime video sources.\n")
		fmt.Fprintf(os.Stderr, "  search          Search for anime on a source.\n")
		fmt.Fprintf(os.Stderr, "  source-info     Get information about a specific anime video source.\n")
		fmt.Fprintf(os.Stderr, "  stream-url      Get the direct video stream URL for an anime episode.\n")
	}

	// Parse flags after the command
	args := os.Args[1:]
	if len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	command := args[0]
	flag.CommandLine.Parse(args[1:])

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	s := NewAllanimeScaper()

	var result interface{}
	var err error

	switch command {
	case "extension-info":
		result, err = s.GetExtensionInfo()

	case "list-sources":
		// Get extension info and return just the sources
		info, err := s.GetExtensionInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting extension info: %v\n", err)
			os.Exit(1)
		}
		result = info.Sources

	case "source-info":
		// If a specific source ID is provided, verify it matches our source
		if *sourceID != "" && *sourceID != "3160569130087668532" {
			fmt.Fprintf(os.Stderr, "Error: invalid source ID %q\n", *sourceID)
			os.Exit(1)
		}
		result, err = s.GetSourceInfo()

	case "search":
		if *query == "" {
			fmt.Fprintf(os.Stderr, "Error: search query is required\n")
			os.Exit(1)
		}
		// If a specific source ID is provided, verify it matches our source
		if *sourceID != "" && *sourceID != "3160569130087668532" {
			fmt.Fprintf(os.Stderr, "Error: invalid source ID %q\n", *sourceID)
			os.Exit(1)
		}
		result, err = s.SearchAnime(*query, *page, *filters)

	case "episodes":
		if *animeURL == "" {
			fmt.Fprintf(os.Stderr, "Error: anime URL is required\n")
			os.Exit(1)
		}
		// If a specific source ID is provided, verify it matches our source
		if *sourceID != "" && *sourceID != "3160569130087668532" {
			fmt.Fprintf(os.Stderr, "Error: invalid source ID %q\n", *sourceID)
			os.Exit(1)
		}
		result, err = s.GetEpisodeList(*animeURL)

	case "stream-url":
		if *animeURL == "" || *episode == 0 {
			fmt.Fprintf(os.Stderr, "Error: anime URL and episode number are required\n")
			os.Exit(1)
		}
		// If a specific source ID is provided, verify it matches our source
		if *sourceID != "" && *sourceID != "3160569130087668532" {
			fmt.Fprintf(os.Stderr, "Error: invalid source ID %q\n", *sourceID)
			os.Exit(1)
		}
		result, err = s.GetVideoList(*animeURL, *episode)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", command)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output the result as JSON
	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling result to JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonOutput))
}
