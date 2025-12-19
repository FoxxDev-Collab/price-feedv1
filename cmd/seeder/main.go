package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
)

const (
	// US zip code data from scpike/us-state-county-zip
	zipCodeDataURL = "https://raw.githubusercontent.com/scpike/us-state-county-zip/master/geo-data.csv"
)

// CityData holds aggregated zip codes for a city
type CityData struct {
	Name     string
	State    string
	County   string
	ZipCodes []string
}

func main() {
	// Command line flags
	dryRun := flag.Bool("dry-run", false, "Preview changes without writing to database")
	minZips := flag.Int("min-zips", 1, "Minimum zip codes required for a city to be included")
	stateFilter := flag.String("state", "", "Only import cities from this state (e.g., 'CO')")
	localFile := flag.String("file", "", "Use local CSV file instead of downloading")
	flag.Parse()

	// Load .env
	godotenv.Load()

	// Load config
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Starting zip code data import...")

	// Get CSV data
	var reader io.Reader
	if *localFile != "" {
		file, err := os.Open(*localFile)
		if err != nil {
			log.Fatalf("Failed to open local file: %v", err)
		}
		defer file.Close()
		reader = file
		log.Printf("Reading from local file: %s", *localFile)
	} else {
		log.Printf("Downloading zip code data from: %s", zipCodeDataURL)
		resp, err := http.Get(zipCodeDataURL)
		if err != nil {
			log.Fatalf("Failed to download zip code data: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Failed to download: HTTP %d", resp.StatusCode)
		}
		reader = resp.Body
	}

	// Parse CSV and aggregate by city
	cities, err := parseZipCodeData(reader, *stateFilter, *minZips)
	if err != nil {
		log.Fatalf("Failed to parse zip code data: %v", err)
	}

	log.Printf("Found %d cities to import", len(cities))

	if *dryRun {
		log.Println("DRY RUN - No changes will be made")
		printPreview(cities, 20)
		return
	}

	// Import to database
	imported, updated, err := importCities(db, cities)
	if err != nil {
		log.Fatalf("Failed to import cities: %v", err)
	}

	log.Printf("Import complete: %d new cities, %d updated", imported, updated)
}

// parseZipCodeData reads CSV and aggregates zip codes by city
func parseZipCodeData(reader io.Reader, stateFilter string, minZips int) ([]CityData, error) {
	csvReader := csv.NewReader(bufio.NewReader(reader))

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Find column indices
	// Expected: state_fips,state,state_abbr,zipcode,county,city
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	stateCol, ok := colMap["state_abbr"]
	if !ok {
		stateCol = colMap["state"]
	}
	zipCol := colMap["zipcode"]
	cityCol := colMap["city"]
	countyCol := colMap["county"]

	// Aggregate zip codes by city+state
	cityMap := make(map[string]*CityData)
	rowCount := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: skipping malformed row: %v", err)
			continue
		}

		rowCount++

		// Extract fields
		state := strings.TrimSpace(record[stateCol])
		zipCode := strings.TrimSpace(record[zipCol])
		city := strings.TrimSpace(record[cityCol])
		county := ""
		if countyCol < len(record) {
			county = strings.TrimSpace(record[countyCol])
		}

		// Apply state filter if specified
		if stateFilter != "" && strings.ToUpper(state) != strings.ToUpper(stateFilter) {
			continue
		}

		// Skip empty entries
		if city == "" || state == "" || zipCode == "" {
			continue
		}

		// Normalize state to uppercase
		state = strings.ToUpper(state)

		// Create key for city+state
		key := fmt.Sprintf("%s|%s", strings.ToLower(city), state)

		if existing, ok := cityMap[key]; ok {
			// Add zip code if not already present
			if !contains(existing.ZipCodes, zipCode) {
				existing.ZipCodes = append(existing.ZipCodes, zipCode)
			}
		} else {
			cityMap[key] = &CityData{
				Name:     city,
				State:    state,
				County:   county,
				ZipCodes: []string{zipCode},
			}
		}
	}

	log.Printf("Processed %d rows", rowCount)

	// Convert map to slice and filter by min zips
	var cities []CityData
	for _, city := range cityMap {
		if len(city.ZipCodes) >= minZips {
			// Sort zip codes for consistency
			sort.Strings(city.ZipCodes)
			cities = append(cities, *city)
		}
	}

	// Sort by state, then city name
	sort.Slice(cities, func(i, j int) bool {
		if cities[i].State != cities[j].State {
			return cities[i].State < cities[j].State
		}
		return cities[i].Name < cities[j].Name
	})

	return cities, nil
}

// importCities imports city data to the regions table using batched transactions
func importCities(db *database.DB, cities []CityData) (imported, updated int, err error) {
	ctx := context.Background()
	batchSize := 500 // Commit every 500 cities to avoid long transactions

	for i := 0; i < len(cities); i += batchSize {
		end := i + batchSize
		if end > len(cities) {
			end = len(cities)
		}
		batch := cities[i:end]

		batchImported, batchUpdated, err := importBatch(ctx, db, batch)
		if err != nil {
			return imported, updated, err
		}
		imported += batchImported
		updated += batchUpdated

		log.Printf("Progress: %d/%d cities processed (%d new, %d updated)",
			end, len(cities), imported, updated)
	}

	return imported, updated, nil
}

// importBatch imports a batch of cities in a single transaction
func importBatch(ctx context.Context, db *database.DB, cities []CityData) (imported, updated int, err error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, city := range cities {
		// Check if city already exists
		var existingID int
		var existingZips []string
		err := tx.QueryRow(ctx, `
			SELECT id, zip_codes FROM regions
			WHERE LOWER(name) = LOWER($1) AND state = $2
		`, city.Name, city.State).Scan(&existingID, &existingZips)

		if err == pgx.ErrNoRows {
			// Insert new city
			_, err = tx.Exec(ctx, `
				INSERT INTO regions (name, state, zip_codes)
				VALUES ($1, $2, $3)
			`, city.Name, city.State, city.ZipCodes)
			if err != nil {
				return imported, updated, fmt.Errorf("failed to insert %s, %s: %w", city.Name, city.State, err)
			}
			imported++
		} else if err != nil {
			return imported, updated, fmt.Errorf("failed to check existing %s, %s: %w", city.Name, city.State, err)
		} else {
			// Merge zip codes with existing
			merged := mergeZipCodes(existingZips, city.ZipCodes)
			if len(merged) > len(existingZips) {
				_, err = tx.Exec(ctx, `
					UPDATE regions SET zip_codes = $1, updated_at = NOW()
					WHERE id = $2
				`, merged, existingID)
				if err != nil {
					return imported, updated, fmt.Errorf("failed to update %s, %s: %w", city.Name, city.State, err)
				}
				updated++
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return imported, updated, nil
}

// mergeZipCodes combines two zip code slices, removing duplicates
func mergeZipCodes(existing, new []string) []string {
	zipSet := make(map[string]bool)
	for _, z := range existing {
		zipSet[z] = true
	}
	for _, z := range new {
		zipSet[z] = true
	}

	var merged []string
	for z := range zipSet {
		merged = append(merged, z)
	}
	sort.Strings(merged)
	return merged
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// printPreview shows a sample of the data to be imported
func printPreview(cities []CityData, limit int) {
	fmt.Println("\n=== Preview of cities to import ===")
	fmt.Printf("Total: %d cities\n\n", len(cities))

	// Group by state for summary
	stateCount := make(map[string]int)
	for _, city := range cities {
		stateCount[city.State]++
	}

	fmt.Println("Cities per state:")
	states := make([]string, 0, len(stateCount))
	for s := range stateCount {
		states = append(states, s)
	}
	sort.Strings(states)
	for _, s := range states {
		fmt.Printf("  %s: %d cities\n", s, stateCount[s])
	}

	fmt.Printf("\nSample cities (first %d):\n", limit)
	for i, city := range cities {
		if i >= limit {
			break
		}
		fmt.Printf("  %s, %s - %d zip codes (%s...)\n",
			city.Name, city.State, len(city.ZipCodes),
			strings.Join(city.ZipCodes[:min(3, len(city.ZipCodes))], ", "))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
