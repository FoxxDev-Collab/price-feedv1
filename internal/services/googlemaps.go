package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	geocodeAPIURL       = "https://maps.googleapis.com/maps/api/geocode/json"
	placesNearbyAPIURL  = "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	placesTextSearchURL = "https://maps.googleapis.com/maps/api/place/textsearch/json"
	placeDetailsAPIURL  = "https://maps.googleapis.com/maps/api/place/details/json"
	defaultTimeout      = 10 * time.Second
	defaultSearchRadius = 5000 // 5km in meters
)

var (
	ErrNoResults      = errors.New("no results found")
	ErrAPIError       = errors.New("google maps api error")
	ErrInvalidAPIKey  = errors.New("invalid or missing api key")
	ErrRequestDenied  = errors.New("request denied by google api")
	ErrOverQueryLimit = errors.New("over query limit")
	ErrInvalidRequest = errors.New("invalid request")
)

// GoogleMapsService provides methods to interact with Google Maps APIs
type GoogleMapsService struct {
	apiKey     string
	httpClient *http.Client
}

// GeocodingResult represents the result of a geocoding operation
type GeocodingResult struct {
	FormattedAddress string            `json:"formatted_address"`
	Latitude         float64           `json:"latitude"`
	Longitude        float64           `json:"longitude"`
	PlaceID          string            `json:"place_id"`
	Components       AddressComponents `json:"components"`
}

// AddressComponents contains parsed address components
type AddressComponents struct {
	StreetNumber string `json:"street_number,omitempty"`
	Route        string `json:"route,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	StateCode    string `json:"state_code,omitempty"`
	Country      string `json:"country,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
}

// PlaceResult represents a place from the Places API
type PlaceResult struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	FormattedAddress string   `json:"formatted_address"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Types            []string `json:"types"`
	Rating           float64  `json:"rating,omitempty"`
	UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
	OpenNow          *bool    `json:"open_now,omitempty"`
	PriceLevel       *int     `json:"price_level,omitempty"`
}

// PlaceDetails represents detailed information about a place
type PlaceDetails struct {
	PlaceID              string            `json:"place_id"`
	Name                 string            `json:"name"`
	FormattedAddress     string            `json:"formatted_address"`
	FormattedPhoneNumber string            `json:"formatted_phone_number,omitempty"`
	Website              string            `json:"website,omitempty"`
	Latitude             float64           `json:"latitude"`
	Longitude            float64           `json:"longitude"`
	Types                []string          `json:"types"`
	Rating               float64           `json:"rating,omitempty"`
	UserRatingsTotal     int               `json:"user_ratings_total,omitempty"`
	OpenNow              *bool             `json:"open_now,omitempty"`
	PriceLevel           *int              `json:"price_level,omitempty"`
	OpeningHours         []string          `json:"opening_hours,omitempty"`
	Components           AddressComponents `json:"components,omitempty"`
	// Convenience fields for store creation
	StreetAddress string `json:"street_address,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	ZipCode       string `json:"zip_code,omitempty"`
}

// Google API response structures
type geocodeResponse struct {
	Status  string `json:"status"`
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		PlaceID          string `json:"place_id"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
	} `json:"results"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type placesNearbyResponse struct {
	Status  string `json:"status"`
	Results []struct {
		PlaceID  string `json:"place_id"`
		Name     string `json:"name"`
		Vicinity string `json:"vicinity"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Types            []string `json:"types"`
		Rating           float64  `json:"rating,omitempty"`
		UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
		OpeningHours     *struct {
			OpenNow bool `json:"open_now"`
		} `json:"opening_hours,omitempty"`
		PriceLevel *int `json:"price_level,omitempty"`
	} `json:"results"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type placeDetailsResponse struct {
	Status string `json:"status"`
	Result struct {
		PlaceID              string `json:"place_id"`
		Name                 string `json:"name"`
		FormattedAddress     string `json:"formatted_address"`
		FormattedPhoneNumber string `json:"formatted_phone_number,omitempty"`
		Website              string `json:"website,omitempty"`
		Geometry             struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		Types            []string `json:"types"`
		Rating           float64  `json:"rating,omitempty"`
		UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
		OpeningHours     *struct {
			OpenNow     bool     `json:"open_now"`
			WeekdayText []string `json:"weekday_text"`
		} `json:"opening_hours,omitempty"`
		PriceLevel *int `json:"price_level,omitempty"`
	} `json:"result"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewGoogleMapsService creates a new GoogleMapsService instance
func NewGoogleMapsService(apiKey string) *GoogleMapsService {
	return &GoogleMapsService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Geocode converts an address string to coordinates
func (s *GoogleMapsService) Geocode(ctx context.Context, address string) (*GeocodingResult, error) {
	if s.apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	params := url.Values{}
	params.Set("address", address)
	params.Set("key", s.apiKey)

	reqURL := geocodeAPIURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var geocodeResp geocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geocodeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if err := checkGoogleAPIStatus(geocodeResp.Status, geocodeResp.ErrorMessage); err != nil {
		return nil, err
	}

	if len(geocodeResp.Results) == 0 {
		return nil, ErrNoResults
	}

	result := geocodeResp.Results[0]
	return &GeocodingResult{
		FormattedAddress: result.FormattedAddress,
		Latitude:         result.Geometry.Location.Lat,
		Longitude:        result.Geometry.Location.Lng,
		PlaceID:          result.PlaceID,
		Components:       parseAddressComponents(result.AddressComponents),
	}, nil
}

// ReverseGeocode converts coordinates to an address
func (s *GoogleMapsService) ReverseGeocode(ctx context.Context, lat, lng float64) (*GeocodingResult, error) {
	if s.apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("key", s.apiKey)

	reqURL := geocodeAPIURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var geocodeResp geocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geocodeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if err := checkGoogleAPIStatus(geocodeResp.Status, geocodeResp.ErrorMessage); err != nil {
		return nil, err
	}

	if len(geocodeResp.Results) == 0 {
		return nil, ErrNoResults
	}

	result := geocodeResp.Results[0]
	return &GeocodingResult{
		FormattedAddress: result.FormattedAddress,
		Latitude:         result.Geometry.Location.Lat,
		Longitude:        result.Geometry.Location.Lng,
		PlaceID:          result.PlaceID,
		Components:       parseAddressComponents(result.AddressComponents),
	}, nil
}

// NearbySearch searches for places near a location
// radius is in meters, placeType can be "supermarket", "grocery_or_supermarket", "store", etc.
// If placeType is empty, searches for both supermarket and grocery_store types
func (s *GoogleMapsService) NearbySearch(ctx context.Context, lat, lng float64, radius int, placeType string) ([]*PlaceResult, error) {
	if s.apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if radius <= 0 {
		radius = defaultSearchRadius
	}

	// If no specific type requested, search for supermarkets
	// Note: "grocery_store" is not a valid type in the legacy Places API - it causes
	// the API to ignore the type filter entirely and return all nearby places
	if placeType == "" {
		placeType = "supermarket"
	}

	return s.nearbySearchSingleType(ctx, lat, lng, radius, placeType)
}

// nearbySearchMultipleTypes searches for multiple place types and deduplicates results
func (s *GoogleMapsService) nearbySearchMultipleTypes(ctx context.Context, lat, lng float64, radius int, placeTypes []string) ([]*PlaceResult, error) {
	seen := make(map[string]bool)
	var allPlaces []*PlaceResult

	for _, pType := range placeTypes {
		places, err := s.nearbySearchSingleType(ctx, lat, lng, radius, pType)
		if err != nil {
			// Continue with other types even if one fails
			continue
		}

		for _, place := range places {
			if !seen[place.PlaceID] {
				seen[place.PlaceID] = true
				allPlaces = append(allPlaces, place)
			}
		}
	}

	return allPlaces, nil
}

// nearbySearchSingleType performs a nearby search for a single place type
func (s *GoogleMapsService) nearbySearchSingleType(ctx context.Context, lat, lng float64, radius int, placeType string) ([]*PlaceResult, error) {
	params := url.Values{}
	params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("radius", strconv.Itoa(radius))
	params.Set("key", s.apiKey)
	params.Set("type", placeType)

	reqURL := placesNearbyAPIURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var placesResp placesNearbyResponse
	if err := json.NewDecoder(resp.Body).Decode(&placesResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if err := checkGoogleAPIStatus(placesResp.Status, placesResp.ErrorMessage); err != nil {
		// ZERO_RESULTS is not an error, just means no results found
		if errors.Is(err, ErrNoResults) {
			return []*PlaceResult{}, nil
		}
		return nil, err
	}

	places := make([]*PlaceResult, 0, len(placesResp.Results))
	for _, p := range placesResp.Results {
		place := &PlaceResult{
			PlaceID:          p.PlaceID,
			Name:             p.Name,
			FormattedAddress: p.Vicinity,
			Latitude:         p.Geometry.Location.Lat,
			Longitude:        p.Geometry.Location.Lng,
			Types:            p.Types,
			Rating:           p.Rating,
			UserRatingsTotal: p.UserRatingsTotal,
			PriceLevel:       p.PriceLevel,
		}
		if p.OpeningHours != nil {
			place.OpenNow = &p.OpeningHours.OpenNow
		}
		places = append(places, place)
	}

	return places, nil
}

// GetPlaceDetails retrieves detailed information about a place
func (s *GoogleMapsService) GetPlaceDetails(ctx context.Context, placeID string) (*PlaceDetails, error) {
	if s.apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	params := url.Values{}
	params.Set("place_id", placeID)
	params.Set("fields", "place_id,name,formatted_address,formatted_phone_number,website,geometry,address_components,types,rating,user_ratings_total,opening_hours,price_level")
	params.Set("key", s.apiKey)

	reqURL := placeDetailsAPIURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var detailsResp placeDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&detailsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if err := checkGoogleAPIStatus(detailsResp.Status, detailsResp.ErrorMessage); err != nil {
		return nil, err
	}

	r := detailsResp.Result

	// Parse address components
	components := parseAddressComponents(r.AddressComponents)

	// Build street address from components
	streetAddress := ""
	if components.StreetNumber != "" && components.Route != "" {
		streetAddress = components.StreetNumber + " " + components.Route
	} else if components.Route != "" {
		streetAddress = components.Route
	}

	details := &PlaceDetails{
		PlaceID:              r.PlaceID,
		Name:                 r.Name,
		FormattedAddress:     r.FormattedAddress,
		FormattedPhoneNumber: r.FormattedPhoneNumber,
		Website:              r.Website,
		Latitude:             r.Geometry.Location.Lat,
		Longitude:            r.Geometry.Location.Lng,
		Types:                r.Types,
		Rating:               r.Rating,
		UserRatingsTotal:     r.UserRatingsTotal,
		PriceLevel:           r.PriceLevel,
		Components:           components,
		// Convenience fields for store creation
		StreetAddress: streetAddress,
		City:          components.City,
		State:         components.StateCode,
		ZipCode:       components.PostalCode,
	}

	if r.OpeningHours != nil {
		details.OpenNow = &r.OpeningHours.OpenNow
		details.OpeningHours = r.OpeningHours.WeekdayText
	}

	return details, nil
}

// parseAddressComponents extracts relevant fields from Google's address components
func parseAddressComponents(components []struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}) AddressComponents {
	ac := AddressComponents{}
	for _, c := range components {
		for _, t := range c.Types {
			switch t {
			case "street_number":
				ac.StreetNumber = c.LongName
			case "route":
				ac.Route = c.LongName
			case "locality":
				ac.City = c.LongName
			case "administrative_area_level_1":
				ac.State = c.LongName
				ac.StateCode = c.ShortName
			case "country":
				ac.Country = c.LongName
				ac.CountryCode = c.ShortName
			case "postal_code":
				ac.PostalCode = c.LongName
			}
		}
	}
	return ac
}

// checkGoogleAPIStatus converts Google API status codes to errors
func checkGoogleAPIStatus(status, errorMessage string) error {
	switch status {
	case "OK":
		return nil
	case "ZERO_RESULTS":
		return ErrNoResults
	case "OVER_QUERY_LIMIT", "OVER_DAILY_LIMIT":
		return ErrOverQueryLimit
	case "REQUEST_DENIED":
		if errorMessage != "" {
			return fmt.Errorf("%w: %s", ErrRequestDenied, errorMessage)
		}
		return ErrRequestDenied
	case "INVALID_REQUEST":
		if errorMessage != "" {
			return fmt.Errorf("%w: %s", ErrInvalidRequest, errorMessage)
		}
		return ErrInvalidRequest
	default:
		if errorMessage != "" {
			return fmt.Errorf("%w: %s - %s", ErrAPIError, status, errorMessage)
		}
		return fmt.Errorf("%w: %s", ErrAPIError, status)
	}
}

// placesTextSearchResponse represents the response from the Text Search API
type placesTextSearchResponse struct {
	Status  string `json:"status"`
	Results []struct {
		PlaceID          string `json:"place_id"`
		Name             string `json:"name"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Types            []string `json:"types"`
		Rating           float64  `json:"rating,omitempty"`
		UserRatingsTotal int      `json:"user_ratings_total,omitempty"`
		OpeningHours     *struct {
			OpenNow bool `json:"open_now"`
		} `json:"opening_hours,omitempty"`
		PriceLevel *int `json:"price_level,omitempty"`
	} `json:"results"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TextSearch searches for places by text query with optional location bias
// query is the search string (e.g., "King Soopers", "Safeway grocery")
// lat, lng provide location bias (results closer to this point are prioritized)
// radius is in meters (used for location bias, not strict filtering)
func (s *GoogleMapsService) TextSearch(ctx context.Context, query string, lat, lng float64, radius int) ([]*PlaceResult, error) {
	if s.apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if query == "" {
		return nil, ErrInvalidRequest
	}

	if radius <= 0 {
		radius = defaultSearchRadius
	}

	params := url.Values{}
	params.Set("query", query+" grocery store")
	params.Set("key", s.apiKey)

	// Add location bias if coordinates are provided
	if lat != 0 || lng != 0 {
		params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
		params.Set("radius", strconv.Itoa(radius))
	}

	reqURL := placesTextSearchURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	var textResp placesTextSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&textResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if err := checkGoogleAPIStatus(textResp.Status, textResp.ErrorMessage); err != nil {
		if errors.Is(err, ErrNoResults) {
			return []*PlaceResult{}, nil
		}
		return nil, err
	}

	places := make([]*PlaceResult, 0, len(textResp.Results))
	for _, p := range textResp.Results {
		place := &PlaceResult{
			PlaceID:          p.PlaceID,
			Name:             p.Name,
			FormattedAddress: p.FormattedAddress,
			Latitude:         p.Geometry.Location.Lat,
			Longitude:        p.Geometry.Location.Lng,
			Types:            p.Types,
			Rating:           p.Rating,
			UserRatingsTotal: p.UserRatingsTotal,
			PriceLevel:       p.PriceLevel,
		}
		if p.OpeningHours != nil {
			place.OpenNow = &p.OpeningHours.OpenNow
		}
		places = append(places, place)
	}

	return places, nil
}
