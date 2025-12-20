Feature: Paste Store Product URL to Auto-Fill Price Entry
Summary
Users can paste product URLs from store websites (Walmart, Kroger, Target, etc.) to automatically extract product name, brand, price, and size - reducing manual data entry.
User Flow
User clicks "Add Price" and sees new "Import from URL" section
User pastes a product URL (e.g., walmart.com/ip/Great-Value-Milk-1-gal/12345)
System fetches and parses the page server-side
Extracted data is shown in a preview and auto-fills the form
User reviews/edits, then submits
Architecture

Frontend (paste URL) --> POST /api/prices/parse-url --> Scraper Service
                                                            |
                                                    Store-Specific Parser
                                                            |
                                                    Match existing store/item
                                                            |
                                    <-- Return ProductData + matched entities
Files to Create
1. internal/scraper/scraper.go - Core Service
ProductData struct (name, brand, price, size, unit, store_name)
StoreParser interface for modular parsers
Service with HTTP client and parser registry
ParseURL() method to fetch and parse
2. internal/scraper/walmart.go - Walmart Parser
Parse JSON-LD structured data
Fallback to meta tags and HTML selectors
Extract size/unit from product name
3. internal/scraper/kroger.go - Kroger Family Parser
Supports: kroger.com, kingsoopers.com, ralphs.com, dillons.com
Similar JSON-LD parsing strategy
4. internal/scraper/target.go - Target Parser
Parse __TGT_DATA__ embedded JSON
Files to Modify
1. internal/handlers/prices.go
Add two new handlers:
ParseProductURL - accepts URL, returns extracted ProductData
CreateFromParsed - creates item (if new) and price from parsed data
2. cmd/server/main.go
Add routes:

prices.Post("/parse-url", middleware.AuthRequired(cfg), h.ParseProductURL)
prices.Post("/create-from-url", middleware.AuthRequired(cfg), h.CreateFromParsed)
3. web/user/prices/index.html
Add to modal:
URL input field with "Import" button
Parse preview section showing extracted data
Loading spinner during parsing
Auto-fill existing form fields from parsed data
4. web/js/api.js
Add scraperApi object:

scraperApi.parseURL(url)      // POST /prices/parse-url
scraperApi.createFromParsed() // POST /prices/create-from-url
5. internal/database/item_repo.go
Add FindSimilarItems() for deduplication matching
6. internal/database/store_repo.go
Add SearchStoresByChain() to match parsed store to existing stores
Dependency to Add

go get github.com/PuerkitoBio/goquery
Supported Stores (Initial)
Store	Domains
Walmart	walmart.com
Kroger	kroger.com, kingsoopers.com, ralphs.com
Target	target.com
Error Handling
Unsupported store: "Sorry, we don't support this store yet"
Fetch failed: "Could not reach store website. Please try again or enter manually"
Parse failed: Allow fallback to manual entry with partial data
Implementation Order
Create internal/scraper/ package with base service
Implement Walmart parser (most common)
Add API endpoints to prices handler
Build frontend UI in prices modal
Add Kroger/Target parsers
Add store/item matching logic
