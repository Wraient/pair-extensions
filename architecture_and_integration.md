# Pair Anime Manager - Architecture and Integration

## Overview

Pair is a comprehensive anime watching and tracking CLI tool built in Go that integrates multiple components to provide a seamless anime management experience. This document explains how all the components work together to create a unified system.

## Core Architecture

### Database-Centric Design

The entire application is built around a SQLite database that serves as the central source of truth for all data:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Commands  │────│  SQLite Database │────│  External APIs  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌────────▼────────┐              │
         └──────────────►│  Business Logic │◄─────────────┘
                         │   (pkg/*)       │
                         └─────────────────┘
```

### Component Layers

1. **Data Layer** (`pkg/database/`)
2. **Business Logic Layer** (`pkg/tracker/`, `pkg/scraper/`, `pkg/extension/`)
3. **Application Layer** (`pkg/appcore/`, `pkg/cmd/`)
4. **Interface Layer** (`pkg/ui/`, CLI commands)

## Database Integration (`pkg/database/`)

### Schema Design

The database schema is designed to handle all aspects of anime management:

```sql
-- Core anime information
anime: id, title, original_title, alternative_titles, description, 
       total_episodes, type, year, season, status, genres, thumbnail_url

-- User progress tracking
episodes: id, anime_id, number, title, description, air_date, 
          duration, watched, watch_time, created_at, updated_at

-- External tracker integration
anime_tracking: id, anime_id, tracker, tracker_id, status, score, 
                current_episode, total_episodes, last_updated

-- Extension management
extensions: id, name, package, version, path, enabled, installed_at
sources: id, extension_id, name, source_id, base_url, language, nsfw

-- Configuration storage
config: key, value, updated_at
```

### Database Operations

Each entity has dedicated operations in separate files:

- **`anime.go`**: CRUD operations for anime data
- **`episode.go`**: Episode tracking and progress management
- **`extension.go`**: Extension and source management
- **`config.go`**: Configuration key-value storage
- **`migrations.go`**: Schema versioning and updates

### Migration System

```go
// Automatic migration on startup
func (db *DB) RunMigrations() error {
    // Check current version
    // Apply pending migrations
    // Update schema version
}
```

## Tracker Integration (`pkg/tracker/`)

### Multi-Tracker Architecture

The tracker system supports multiple external services through a unified interface:

```go
type Tracker interface {
    Name() string
    IsAuthenticated() bool
    Authenticate(ctx context.Context) error
    SearchAnime(ctx context.Context, query string, limit int) ([]AnimeInfo, error)
    SyncFromRemote(ctx context.Context, db *database.DB) (SyncStats, error)
    SyncToRemote(ctx context.Context, db *database.DB) (SyncStats, error)
}
```

### Supported Trackers

1. **Local Tracker** (`local.go`)
   - Stores all data locally in SQLite
   - No external dependencies
   - Default fallback option

2. **MyAnimeList Tracker** (`mal.go`)
   - OAuth2 authentication with MAL
   - Bidirectional synchronization
   - Rate-limited API calls

3. **AniList Tracker** (`anilist.go`)
   - GraphQL API integration
   - OAuth2 authentication
   - Advanced querying capabilities

### Synchronization Flow

```
┌──────────────┐    Sync From Remote    ┌─────────────────┐
│ External API │ ───────────────────────►│ Local Database  │
│ (MAL/AniList)│                         │                 │
│              │ ◄───────────────────────│                 │
└──────────────┘    Sync To Remote      └─────────────────┘
```

**From Remote to Local:**
1. Fetch user's anime list from external tracker
2. Compare with local database entries
3. Add new anime and tracking data
4. Update existing entries if remote is newer
5. Store last sync timestamp

**From Local to Remote:**
1. Get locally modified tracking data since last sync
2. Update external tracker via API
3. Handle rate limits and errors
4. Update sync timestamp on success

## Extension System (`pkg/extension/`)

### CLI-Based Extension Architecture

Extensions are external binaries that implement a standardized CLI interface:

```
┌─────────────────┐    CLI Calls    ┌─────────────────┐
│ Extension       │ ◄───────────────│ Pair Application│
│ Manager         │                 │                 │
│                 │    JSON Response│                 │
│ (Go Binary)     │ ────────────────►│                 │
└─────────────────┘                 └─────────────────┘
```

### Extension Operations

Extensions provide these capabilities:
- **Search**: Find anime by query with pagination
- **Popular**: Get trending/popular anime
- **Latest**: Get recently updated anime  
- **Episodes**: List episodes for an anime
- **Streams**: Get video URLs for episodes
- **Details**: Get detailed anime information

### Extension Management

```go
type CLIManager struct {
    extensionDir string
    db          *database.DB
}

// Extension lifecycle
func (m *CLIManager) Install(repoURL string) error
func (m *CLIManager) Remove(pkg string) error  
func (m *CLIManager) List() ([]ExtensionInfo, error)
func (m *CLIManager) Update(pkg string) error
```

### Database Integration

Extensions are tracked in the database:
- Installation metadata (version, path, enabled status)
- Available sources per extension
- Source capabilities (search, latest, etc.)

## Scraper Integration (`pkg/scraper/`)

### Unified Scraping Interface

The scraper system provides a unified interface to interact with multiple anime sources:

```go
type ScraperRunner struct {
    scrapers map[string]Scraper
    db       *database.DB
}

// Core scraping operations
func (r *ScraperRunner) SearchAnime(sourceID, query string, page int) ([]Anime, error)
func (r *ScraperRunner) GetEpisodes(sourceID, animeID string) ([]Episode, error)
func (r *ScraperRunner) GetStreams(sourceID, animeID string, episode float64) (VideoResponse, error)
```

### Database-Aware Progress Tracking

The scraper integrates with the database for progress tracking:

```go
// Automatic progress updates
func (r *ScraperRunner) UpdateWatchProgress(animeID string, episodeNum float64, watchTime float64) error {
    // Update episode watch time
    // Mark episode as watched if completed
    // Update anime progress tracking
    // Sync with external trackers if enabled
}
```

### Source Management

Sources are dynamically registered from installed extensions:

```go
// During startup
for _, extension := range installedExtensions {
    for _, source := range extension.Sources {
        scraper := NewCLIScraper(extension.Path, source.ID)
        runner.RegisterScraper(source.ID, scraper)
    }
}
```

## Application Core (`pkg/appcore/`)

### Centralized Application Management

The app core orchestrates all components:

```go
type App struct {
    uiManager      ui.UIManager
    db             *database.DB
    extManager     *extension.CLIManager
    scraper        *scraper.ScraperRunner
    trackerManager *tracker.TrackerManager
}

func NewApp() (*App, error) {
    // 1. Initialize database connection
    // 2. Run migrations
    // 3. Initialize extension manager
    // 4. Setup tracker manager with all trackers
    // 5. Initialize scraper runner
    // 6. Setup UI manager
}
```

### Dependency Injection

All components receive database connections through dependency injection:

```go
// Database is initialized once and passed to all components
db := config.GetDB()
extManager := extension.NewCLIManager(extDir, db)
trackerManager := tracker.NewTrackerManager(db)
scraperRunner := scraper.NewScraperRunner(db)
```

## Command Line Interface (`pkg/cmd/`)

### Command Structure

Commands are organized hierarchically:

```
pair
├── continue                 # Continue last watched anime
├── watching                 # Manage currently watching anime
├── list                     # Show all anime
├── extension
│   ├── list                # List installed extensions
│   ├── install <repo>      # Install extension
│   └── remove <pkg>        # Remove extension
├── scraper
│   ├── search <query>      # Search anime
│   ├── episodes <anime>    # Get episode list
│   └── stream <anime> <ep> # Get stream URLs
└── tracker
    ├── sync                # Sync with external trackers
    ├── auth <tracker>      # Authenticate with tracker
    └── status              # Show sync status
```

### Database Integration in Commands

Each command properly initializes the database:

```go
func initializeExtensionManager(extDir string) (*extension.CLIManager, error) {
    db := config.GetDB()
    if db == nil {
        return nil, fmt.Errorf("failed to get database connection")
    }
    return extension.NewCLIManager(extDir, db)
}
```

## Configuration Management (`pkg/config/`)

### Unified Configuration

Configuration is stored both in files and database:

```go
type Config struct {
    Database struct {
        Path string `mapstructure:"path"`
    } `mapstructure:"database"`
    
    Extensions struct {
        Directory string `mapstructure:"directory"`
    } `mapstructure:"extensions"`
    
    Development bool `mapstructure:"development"`
}
```

### Database-Backed Settings

Runtime settings are stored in the database:
- Active tracker selection
- Last sync timestamps
- User preferences
- Theme settings

## User Interface (`pkg/ui/`)

### Multi-Modal Interface

The UI system supports multiple interaction modes:

1. **TUI (Terminal UI)** - Interactive terminal interface using Bubble Tea
2. **Rofi Integration** - External rofi-based menu system
3. **CLI Commands** - Direct command-line operations

### State Management

UI components interact with the database through the business logic layer:

```go
type UIManager interface {
    ShowAnimeList(filter string) error
    ShowEpisodeList(animeID string) error
    PlayEpisode(animeID string, episode float64) error
    UpdateProgress(animeID string, episode float64, progress float64) error
}
```

## Data Flow Examples

### Adding New Anime

1. **User searches** via CLI: `pair scraper search "Jujutsu Kaisen"`
2. **Scraper queries** extensions for matching anime
3. **User selects** anime from results
4. **Database stores** anime metadata
5. **Tracker syncs** with external services if configured

### Watching an Episode

1. **User requests** episode: `pair scraper stream <anime-id> 1`
2. **Scraper fetches** stream URLs from extension
3. **Video player** launched with stream URL
4. **Progress tracked** in database during playback
5. **Episode marked** as watched when completed
6. **External trackers** updated automatically

### Synchronizing with MAL

1. **User triggers** sync: `pair tracker sync`
2. **MAL tracker** fetches user's anime list
3. **Database compares** local vs remote data
4. **Conflicts resolved** (remote takes precedence)
5. **Local changes** pushed to MAL
6. **Sync timestamp** updated

## Error Handling and Resilience

### Database Error Handling

All database operations use standard Go error handling with `sql.ErrNoRows` for missing records:

```go
anime, err := db.GetAnime(id)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        // Handle missing anime
    } else {
        // Handle database error
    }
}
```

### External API Resilience

- Automatic token refresh for OAuth2 trackers
- Rate limiting and backoff for API calls  
- Graceful degradation when services unavailable
- Local fallback for all operations

### Extension Error Handling

- Validation of extension responses
- Fallback to other sources on failure
- Extension health monitoring
- Automatic retry mechanisms

## Performance Considerations

### Database Optimization

- Proper indexing on frequently queried columns
- Prepared statements for repeated queries
- Transaction batching for bulk operations
- Connection pooling for concurrent access

### Caching Strategy

- Extension metadata cached in database
- Anime search results cached temporarily
- Stream URLs cached for session duration
- Tracker data cached between syncs

### Concurrent Operations

- Goroutines for parallel extension queries
- Context-based cancellation for long operations
- Worker pools for batch processing
- Rate limiting for external APIs

## Future Extensibility

### Plugin Architecture

The extension system is designed for easy expansion:
- New anime sources via extension binaries
- Custom tracker implementations
- Additional output formats
- Enhanced UI modes

### API Extensibility  

The database schema supports future enhancements:
- Additional metadata fields
- New tracking states
- Extended progress metrics
- Custom user data

### Integration Points

Well-defined interfaces allow integration with:
- External media players
- Download managers
- Notification systems
- Web interfaces
- Mobile applications

## Conclusion

Pair's architecture successfully integrates multiple complex systems through a database-centric design. The SQLite database serves as the authoritative source of truth, while well-defined interfaces enable modular components to work together seamlessly. This design provides both robustness and flexibility for current needs while maintaining extensibility for future enhancements.
