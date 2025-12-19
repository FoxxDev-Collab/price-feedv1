# Price Feed - Complete Implementation Plan

## Executive Summary

**Project:** Price Feed - Community-Driven Grocery Price Comparison Platform
**Owner:** Foxx Cyber LLC (Jeremiah Price)
**Target Market:** Colorado Springs, CO (MVP) → National expansion
**Timeline:** 12-16 weeks to MVP launch
**Tech Stack:** Go (API + static file server), Vanilla HTML/CSS/JS, PostgreSQL, VPS hosting

> **Architecture Decision:** We use vanilla HTML/CSS/JS instead of a JavaScript framework. Go serves both the API and static files. No npm, no build step, no dependency hell. Simple, fast, and maintainable.  

---

## Phase 1: Foundation & Infrastructure (Weeks 1-2)

### 1.1 Development Environment Setup

**Local Development:**
```bash
# Required installations
- Go 1.25+
- PostgreSQL 16
- VS Code with Go extension
- Browser DevTools (for frontend debugging)
```

**Version Control:**
- GitHub repository (private)
- Branch strategy: `main`, `develop`, `feature/*`
- Commit message convention: Conventional Commits

**Project Structure:**
```
price-feed/
├── cmd/
│   └── server/
│       └── main.go          # Entry point - serves API + static files
├── internal/
│   ├── config/
│   ├── database/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── services/
│   └── utils/
├── migrations/
├── web/                     # Static frontend (served by Go)
│   ├── index.html
│   ├── css/
│   │   ├── styles.css
│   │   └── components.css
│   ├── js/
│   │   ├── app.js           # Main application logic
│   │   ├── api.js           # API client (fetch wrapper)
│   │   ├── auth.js          # Authentication handling
│   │   ├── stores.js        # Store management
│   │   ├── items.js         # Item management
│   │   ├── lists.js         # Shopping list logic
│   │   └── utils.js         # Utility functions
│   ├── pages/
│   │   ├── login.html
│   │   ├── register.html
│   │   ├── dashboard.html
│   │   ├── stores.html
│   │   ├── items.html
│   │   └── lists.html
│   └── assets/
│       ├── images/
│       └── favicon.ico
├── go.mod
├── go.sum
└── README.md
```

### 1.2 VPS Setup

**Server Specifications:**
- Provider: Linode/Hetzner/DigitalOcean
- Specs: 4 CPU, 8GB RAM, 100GB SSD
- OS: Ubuntu 24.04 LTS
- Cost: ~$40/month

**Initial Server Configuration:**
```bash
# SSH hardening
- Disable root login
- Setup SSH keys
- Configure UFW firewall (22, 80, 443)
- Install fail2ban

# Install required software
apt update && apt upgrade -y
apt install -y nginx postgresql-16 certbot python3-certbot-nginx

# Install Go
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz

# Setup systemd service (single service serves API + static files)
# price-feed.service
```

**Domain & SSL:**
- Register domain: pricefeed.foxxcyber.com
- Point DNS to VPS IP
- Setup Cloudflare proxy (free tier)
- Configure SSL with Let's Encrypt

### 1.3 Database Design

**Complete Schema:**

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For fuzzy search

-- Regions table
CREATE TABLE regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    zip_codes TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    username VARCHAR(50) UNIQUE,
    region_id INT REFERENCES regions(id),
    reputation_points INT DEFAULT 0,
    role VARCHAR(20) DEFAULT 'user', -- user, admin, moderator
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);

-- Stores table
CREATE TABLE stores (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    street_address VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    region_id INT REFERENCES regions(id) NOT NULL,
    store_type VARCHAR(50), -- grocery, supermarket, warehouse, etc.
    chain VARCHAR(100), -- kroger, walmart, albertsons, etc.
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    verified BOOLEAN DEFAULT FALSE,
    verification_count INT DEFAULT 0,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_store_address UNIQUE (
        LOWER(TRIM(street_address)), 
        UPPER(state), 
        zip_code, 
        region_id
    )
);

-- Items table
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(100),
    size DECIMAL(10, 3) NOT NULL,
    unit VARCHAR(20) NOT NULL, -- gallon, oz, lb, count, etc.
    description TEXT,
    verified BOOLEAN DEFAULT FALSE,
    verification_count INT DEFAULT 0,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tags table
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Item-Tag junction
CREATE TABLE item_tags (
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (item_id, tag_id)
);

-- Store prices table
CREATE TABLE store_prices (
    id SERIAL PRIMARY KEY,
    store_id INT REFERENCES stores(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    price DECIMAL(10, 2) NOT NULL,
    user_id INT REFERENCES users(id),
    verified_count INT DEFAULT 0,
    last_verified TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Price verifications table
CREATE TABLE price_verifications (
    id SERIAL PRIMARY KEY,
    price_id INT REFERENCES store_prices(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id),
    is_accurate BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_user_price_verification UNIQUE (price_id, user_id)
);

-- Shopping lists table
CREATE TABLE shopping_lists (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    target_date DATE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Shopping list items table
CREATE TABLE shopping_list_items (
    id SERIAL PRIMARY KEY,
    list_id INT REFERENCES shopping_lists(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id),
    quantity INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Store plans table (generated optimizations)
CREATE TABLE store_plans (
    id SERIAL PRIMARY KEY,
    list_id INT REFERENCES shopping_lists(id) ON DELETE CASCADE,
    total_savings DECIMAL(10, 2),
    recommended_strategy TEXT, -- single-store, multi-store
    generated_at TIMESTAMP DEFAULT NOW()
);

-- Store plan items table
CREATE TABLE store_plan_items (
    id SERIAL PRIMARY KEY,
    plan_id INT REFERENCES store_plans(id) ON DELETE CASCADE,
    store_id INT REFERENCES stores(id),
    item_id INT REFERENCES items(id),
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Price feed table (activity feed)
CREATE TABLE price_feed (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    store_id INT REFERENCES stores(id),
    item_id INT REFERENCES items(id),
    price DECIMAL(10, 2),
    action VARCHAR(50), -- 'updated', 'verified', 'added', 'hot_deal'
    region_id INT REFERENCES regions(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- User sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_stores_region ON stores(region_id);
CREATE INDEX idx_stores_zip ON stores(zip_code);
CREATE INDEX idx_store_prices_store ON store_prices(store_id);
CREATE INDEX idx_store_prices_item ON store_prices(item_id);
CREATE INDEX idx_store_prices_updated ON store_prices(updated_at DESC);
CREATE INDEX idx_items_name_trgm ON items USING gin(name gin_trgm_ops);
CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_tags_usage ON tags(usage_count DESC);
CREATE INDEX idx_price_feed_region ON price_feed(region_id, created_at DESC);
CREATE INDEX idx_price_feed_user ON price_feed(user_id, created_at DESC);

-- Create address normalization function
CREATE OR REPLACE FUNCTION normalize_address(addr TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(
        REGEXP_REPLACE(
            REGEXP_REPLACE(
                REGEXP_REPLACE(addr, '\bstreet\b', 'st', 'gi'),
                '\bavenue\b', 'ave', 'gi'
            ),
            '\bboulevard\b', 'blvd', 'gi'
        )
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

**Migration Strategy:**
- Use golang-migrate or similar
- Numbered migration files: `001_initial_schema.up.sql`
- Always create `.up.sql` and `.down.sql` pairs
- Test rollback scenarios

---

## Phase 2: Backend API Development (Weeks 3-6)

### 2.1 Core API Structure

**Technology Stack:**
- Framework: Fiber (high-performance Go web framework)
- Database: pgx (PostgreSQL driver)
- Authentication: JWT tokens
- Validation: go-playground/validator
- Configuration: viper

**Project Dependencies:**
```go
// go.mod
module github.com/foxxcyber/price-feed

go 1.23

require (
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/jackc/pgx/v5 v5.5.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/go-playground/validator/v10 v10.16.0
    github.com/spf13/viper v1.18.2
    golang.org/x/crypto v0.18.0
    github.com/google/uuid v1.5.0
)
```

### 2.2 API Endpoints Specification

**Authentication Endpoints:**
```
POST   /api/auth/register
POST   /api/auth/login
POST   /api/auth/logout
GET    /api/auth/me
POST   /api/auth/refresh
POST   /api/auth/forgot-password
POST   /api/auth/reset-password
```

**User Endpoints:**
```
GET    /api/users/:id
PUT    /api/users/:id
GET    /api/users/:id/stats
GET    /api/users/:id/reputation
```

**Region Endpoints:**
```
GET    /api/regions
GET    /api/regions/:id
POST   /api/regions (admin only)
```

**Store Endpoints:**
```
GET    /api/stores
GET    /api/stores/:id
POST   /api/stores
PUT    /api/stores/:id
DELETE /api/stores/:id
POST   /api/stores/check-duplicate
POST   /api/stores/:id/verify
GET    /api/stores/search?q=&zip=
```

**Item Endpoints:**
```
GET    /api/items
GET    /api/items/:id
POST   /api/items
PUT    /api/items/:id
DELETE /api/items/:id
POST   /api/items/check-duplicate
POST   /api/items/:id/verify
GET    /api/items/search?q=&tags=
```

**Tag Endpoints:**
```
GET    /api/tags
GET    /api/tags/autocomplete?q=
GET    /api/tags/popular
POST   /api/tags
GET    /api/tags/:id/items
```

**Price Endpoints:**
```
GET    /api/prices
POST   /api/prices
PUT    /api/prices/:id
DELETE /api/prices/:id
POST   /api/prices/:id/verify
GET    /api/prices/by-store/:store_id
GET    /api/prices/by-item/:item_id
GET    /api/prices/history/:item_id/:store_id
```

**Shopping List Endpoints:**
```
GET    /api/lists
GET    /api/lists/:id
POST   /api/lists
PUT    /api/lists/:id
DELETE /api/lists/:id
POST   /api/lists/:id/items
DELETE /api/lists/:id/items/:item_id
POST   /api/lists/:id/optimize
GET    /api/lists/:id/store-plan
```

**Feed Endpoints:**
```
GET    /api/feed?region_id=&limit=&offset=
GET    /api/feed/hot-deals?region_id=
GET    /api/feed/user/:user_id
```

### 2.3 Core Business Logic Implementation

**Price Optimization Algorithm:**

```go
type StoreOptimization struct {
    SingleStore    *SingleStoreResult
    MultiStore     *MultiStoreResult
    Recommendation string
}

type SingleStoreResult struct {
    StoreID      int
    StoreName    string
    TotalCost    float64
    ItemsFound   int
    ItemsMissing []string
}

type MultiStoreResult struct {
    Stores       []StoreBreakdown
    TotalCost    float64
    TotalSavings float64
    TripCount    int
}

func OptimizeShoppingList(listID int) (*StoreOptimization, error) {
    // 1. Get all items in the list
    items := GetListItems(listID)
    
    // 2. Get latest prices for each item across all stores
    priceMatrix := BuildPriceMatrix(items)
    
    // 3. Calculate single-store option (cheapest overall)
    singleStore := FindBestSingleStore(priceMatrix)
    
    // 4. Calculate multi-store optimization
    multiStore := OptimizeMultiStore(priceMatrix)
    
    // 5. Determine recommendation
    recommendation := DetermineRecommendation(singleStore, multiStore)
    
    return &StoreOptimization{
        SingleStore:    singleStore,
        MultiStore:     multiStore,
        Recommendation: recommendation,
    }, nil
}

func DetermineRecommendation(single *SingleStoreResult, multi *MultiStoreResult) string {
    savingsThreshold := 10.00 // User configurable
    
    if multi.TotalSavings < savingsThreshold {
        return "single_store" // Convenience wins
    }
    
    if multi.TripCount > 2 {
        return "single_store" // Too many stops
    }
    
    return "multi_store" // Savings worth it
}
```

**Address Normalization Service:**

```go
type AddressService struct {
    db *pgxpool.Pool
}

func (s *AddressService) NormalizeAddress(address string) string {
    normalized := strings.ToLower(strings.TrimSpace(address))
    
    replacements := map[string]string{
        " street":    " st",
        " avenue":    " ave",
        " boulevard": " blvd",
        " drive":     " dr",
        " road":      " rd",
        " parkway":   " pkwy",
        " circle":    " cir",
        " court":     " ct",
        " lane":      " ln",
        " north":     " n",
        " south":     " s",
        " east":      " e",
        " west":      " w",
    }
    
    for full, abbr := range replacements {
        normalized = strings.ReplaceAll(normalized, full, abbr)
    }
    
    // Remove punctuation
    normalized = regexp.MustCompile(`[.,]`).ReplaceAllString(normalized, "")
    
    // Collapse spaces
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
    
    return strings.TrimSpace(normalized)
}

func (s *AddressService) CheckDuplicate(req StoreCreateRequest) (*DuplicateCheckResult, error) {
    normalized := s.NormalizeAddress(req.StreetAddress)
    
    // Check exact match
    var existingStore Store
    err := s.db.QueryRow(context.Background(), `
        SELECT * FROM stores 
        WHERE normalize_address(street_address) = normalize_address($1)
        AND UPPER(state) = UPPER($2)
        AND zip_code = $3
        AND region_id = $4
    `, req.StreetAddress, req.State, req.ZipCode, req.RegionID).Scan(&existingStore)
    
    if err == nil {
        return &DuplicateCheckResult{
            IsDuplicate: true,
            ExactMatch:  &existingStore,
        }, nil
    }
    
    // Find similar stores
    similar := s.FindSimilarStores(req)
    
    return &DuplicateCheckResult{
        IsDuplicate:   false,
        SimilarStores: similar,
    }, nil
}
```

**Tag Management Service:**

```go
func (s *TagService) GetOrCreateTag(name string) (*Tag, error) {
    normalized := normalizeTag(name)
    
    // Try to get existing
    var tag Tag
    err := s.db.QueryRow(context.Background(), `
        SELECT * FROM tags WHERE slug = $1
    `, normalized).Scan(&tag)
    
    if err == nil {
        return &tag, nil
    }
    
    // Create new tag
    err = s.db.QueryRow(context.Background(), `
        INSERT INTO tags (name, slug, usage_count)
        VALUES ($1, $2, 0)
        RETURNING *
    `, name, normalized).Scan(&tag)
    
    return &tag, err
}

func (s *TagService) AutocompleteTags(query string, limit int) ([]TagSuggestion, error) {
    rows, err := s.db.Query(context.Background(), `
        SELECT name, usage_count 
        FROM tags 
        WHERE name ILIKE $1
        ORDER BY usage_count DESC
        LIMIT $2
    `, query+"%", limit)
    
    // Parse results...
    return suggestions, err
}

func normalizeTag(tag string) string {
    normalized := strings.ToLower(strings.TrimSpace(tag))
    normalized = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(normalized, "-")
    normalized = regexp.MustCompile(`-+`).ReplaceAllString(normalized, "-")
    return strings.Trim(normalized, "-")
}
```

### 2.4 Static File Serving (Go)

> **Key Architecture:** The Go server handles both API routes and static files. No separate frontend server needed.

```go
// cmd/server/main.go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
    app := fiber.New()

    // Middleware
    app.Use(logger.New())
    app.Use(cors.New())

    // API routes (under /api prefix)
    api := app.Group("/api")
    api.Post("/auth/register", handlers.Register)
    api.Post("/auth/login", handlers.Login)
    api.Get("/auth/me", middleware.AuthRequired(), handlers.GetCurrentUser)
    // ... other API routes

    // Static files - serve the web/ directory
    // This handles all non-API routes
    app.Static("/", "./web", fiber.Static{
        Index:         "index.html",
        Browse:        false,
        CacheDuration: 24 * time.Hour,
    })

    // Fallback for SPA-style routing (optional)
    app.Get("/*", func(c *fiber.Ctx) error {
        return c.SendFile("./web/index.html")
    })

    log.Fatal(app.Listen(":8080"))
}
```

**Benefits of this approach:**
- Single binary deployment
- No CORS issues (same origin)
- Simpler infrastructure
- No npm/Node.js required
- Hot reload during development with tools like `air`

### 2.5 Middleware Implementation

**Authentication Middleware:**
```go
func AuthRequired() fiber.Handler {
    return func(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        if token == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "unauthorized",
            })
        }
        
        // Validate JWT
        claims, err := ValidateToken(token)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{
                "error": "invalid_token",
            })
        }
        
        // Set user context
        c.Locals("user_id", claims.UserID)
        c.Locals("user_role", claims.Role)
        
        return c.Next()
    }
}
```

**Rate Limiting:**
```go
func RateLimiter() fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        100,
        Expiration: 1 * time.Minute,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
        LimitReached: func(c *fiber.Ctx) error {
            return c.Status(429).JSON(fiber.Map{
                "error": "rate_limit_exceeded",
            })
        },
    })
}
```

---

## Phase 3: Frontend Development (Weeks 5-8)

### 3.1 Vanilla Frontend Architecture

> **Philosophy:** No build step. No npm. No node_modules. Just HTML, CSS, and JavaScript that runs directly in the browser. Go serves these files statically.

```
web/
├── index.html                 # Landing page
├── css/
│   ├── styles.css             # Main stylesheet
│   ├── components.css         # Reusable component styles
│   └── utilities.css          # Utility classes
├── js/
│   ├── app.js                 # App initialization, routing
│   ├── api.js                 # Fetch wrapper for API calls
│   ├── auth.js                # Login, register, session management
│   ├── stores.js              # Store CRUD operations
│   ├── items.js               # Item CRUD operations
│   ├── lists.js               # Shopping list logic
│   ├── feed.js                # Activity feed logic
│   ├── tags.js                # Tag autocomplete
│   └── utils.js               # DOM helpers, formatting, etc.
├── pages/
│   ├── login.html
│   ├── register.html
│   ├── dashboard.html
│   ├── stores/
│   │   ├── index.html         # Store list
│   │   └── new.html           # Add store form
│   ├── items/
│   │   ├── index.html         # Item list
│   │   └── new.html           # Add item form
│   └── lists/
│       ├── index.html         # Shopping lists
│       └── new.html           # Create list
├── components/                # Reusable HTML snippets (loaded via JS)
│   ├── navbar.html
│   ├── footer.html
│   ├── store-card.html
│   ├── item-card.html
│   └── modal.html
└── assets/
    ├── images/
    └── favicon.ico
```

### 3.2 Key Frontend Components

**API Client (Vanilla JS):**
```javascript
// js/api.js
const API_BASE = '/api';

const api = {
  // Get auth token from localStorage
  getToken() {
    return localStorage.getItem('token');
  },

  // Make authenticated request
  async request(endpoint, options = {}) {
    const token = this.getToken();
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    const response = await fetch(`${API_BASE}${endpoint}`, config);

    // Handle 401 - redirect to login
    if (response.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/pages/login.html';
      return;
    }

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },

  get(endpoint) {
    return this.request(endpoint, { method: 'GET' });
  },

  post(endpoint, data) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  put(endpoint, data) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  delete(endpoint) {
    return this.request(endpoint, { method: 'DELETE' });
  },
};
```

**Store Management (Vanilla JS):**
```javascript
// js/stores.js
const stores = {
  async getAll(regionId) {
    const params = regionId ? `?region_id=${regionId}` : '';
    return api.get(`/stores${params}`);
  },

  async getById(id) {
    return api.get(`/stores/${id}`);
  },

  async create(data) {
    return api.post('/stores', data);
  },

  async checkDuplicate(data) {
    return api.post('/stores/check-duplicate', data);
  },

  async verify(id) {
    return api.post(`/stores/${id}/verify`);
  },

  // Render store list to DOM
  renderList(storeData, container) {
    container.innerHTML = storeData.map(store => `
      <div class="store-card" data-id="${store.id}">
        <h3>${escapeHtml(store.name)}</h3>
        <p>${escapeHtml(store.street_address)}</p>
        <p>${escapeHtml(store.city)}, ${store.state} ${store.zip_code}</p>
        ${store.verified ? '<span class="badge verified">Verified</span>' : ''}
        <button onclick="stores.viewDetails(${store.id})">View Prices</button>
      </div>
    `).join('');
  },
};
```

**Tag Input Component (Vanilla JS):**
```javascript
// js/tags.js
function initTagInput(inputId, containerId, hiddenInputId) {
  const input = document.getElementById(inputId);
  const container = document.getElementById(containerId);
  const hiddenInput = document.getElementById(hiddenInputId);
  const selectedTags = [];
  let debounceTimer;

  // Create suggestions dropdown
  const dropdown = document.createElement('div');
  dropdown.className = 'suggestions-dropdown hidden';
  input.parentNode.appendChild(dropdown);

  // Debounced autocomplete
  input.addEventListener('input', (e) => {
    clearTimeout(debounceTimer);
    const query = e.target.value.trim();

    if (query.length < 2) {
      dropdown.classList.add('hidden');
      return;
    }

    debounceTimer = setTimeout(async () => {
      const suggestions = await api.get(`/tags/autocomplete?q=${encodeURIComponent(query)}`);
      renderSuggestions(suggestions);
    }, 300);
  });

  function renderSuggestions(tags) {
    if (!tags.length) {
      dropdown.classList.add('hidden');
      return;
    }

    dropdown.innerHTML = tags.map(tag => `
      <div class="suggestion-item" data-name="${escapeHtml(tag.name)}">
        ${escapeHtml(tag.name)} <span class="count">(${tag.usage_count})</span>
      </div>
    `).join('');
    dropdown.classList.remove('hidden');

    // Add click handlers
    dropdown.querySelectorAll('.suggestion-item').forEach(item => {
      item.addEventListener('click', () => addTag(item.dataset.name));
    });
  }

  function addTag(name) {
    if (!selectedTags.includes(name)) {
      selectedTags.push(name);
      renderSelectedTags();
      updateHiddenInput();
    }
    input.value = '';
    dropdown.classList.add('hidden');
  }

  function removeTag(name) {
    const idx = selectedTags.indexOf(name);
    if (idx > -1) {
      selectedTags.splice(idx, 1);
      renderSelectedTags();
      updateHiddenInput();
    }
  }

  function renderSelectedTags() {
    container.innerHTML = selectedTags.map(tag => `
      <span class="tag-chip">
        ${escapeHtml(tag)}
        <button type="button" onclick="removeTag('${escapeHtml(tag)}')">&times;</button>
      </span>
    `).join('');
  }

  function updateHiddenInput() {
    hiddenInput.value = JSON.stringify(selectedTags);
  }

  // Expose removeTag globally for onclick
  window.removeTag = removeTag;
}
```

### 3.3 State Management Strategy

> **Vanilla JS Approach:** No React, no state libraries. We use simple patterns:
> - localStorage for persistent data (auth token, user preferences)
> - DOM for UI state
> - Simple JS objects for in-memory caching

**Authentication Module:**
```javascript
// js/auth.js
const auth = {
  user: null,

  // Check if user is logged in on page load
  async init() {
    const token = localStorage.getItem('token');
    if (!token) return null;

    try {
      this.user = await api.get('/auth/me');
      this.updateUI();
      return this.user;
    } catch (err) {
      this.logout();
      return null;
    }
  },

  async login(email, password) {
    const response = await api.post('/auth/login', { email, password });
    localStorage.setItem('token', response.token);
    this.user = response.user;
    window.location.href = '/pages/dashboard.html';
  },

  async register(email, password, username) {
    const response = await api.post('/auth/register', { email, password, username });
    localStorage.setItem('token', response.token);
    this.user = response.user;
    window.location.href = '/pages/dashboard.html';
  },

  logout() {
    localStorage.removeItem('token');
    this.user = null;
    window.location.href = '/';
  },

  // Update navbar based on auth state
  updateUI() {
    const authButtons = document.getElementById('auth-buttons');
    const userMenu = document.getElementById('user-menu');

    if (this.user) {
      if (authButtons) authButtons.classList.add('hidden');
      if (userMenu) {
        userMenu.classList.remove('hidden');
        userMenu.querySelector('.username').textContent = this.user.username;
      }
    } else {
      if (authButtons) authButtons.classList.remove('hidden');
      if (userMenu) userMenu.classList.add('hidden');
    }
  },

  // Require auth - redirect if not logged in
  requireAuth() {
    if (!localStorage.getItem('token')) {
      window.location.href = '/pages/login.html';
      return false;
    }
    return true;
  },
};

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => auth.init());
```

**Simple Page Router (optional, for SPA-like behavior):**
```javascript
// js/app.js
// For multi-page app, each page handles its own init
// This runs on every page

document.addEventListener('DOMContentLoaded', async () => {
  // Load common components
  await loadComponent('navbar', 'navbar-container');
  await loadComponent('footer', 'footer-container');

  // Init auth
  await auth.init();
});

async function loadComponent(name, containerId) {
  const container = document.getElementById(containerId);
  if (!container) return;

  try {
    const response = await fetch(`/components/${name}.html`);
    container.innerHTML = await response.text();
  } catch (err) {
    console.error(`Failed to load component: ${name}`);
  }
}

// Utility: escape HTML to prevent XSS
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
```

---

## Phase 4: Testing Strategy (Ongoing from Week 3)

### 4.1 Backend Testing

**Unit Tests:**
```go
// tests/services/price_service_test.go
func TestOptimizeShoppingList(t *testing.T) {
    // Setup test data
    listID := createTestShoppingList()
    
    // Run optimization
    result, err := OptimizeShoppingList(listID)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, result.SingleStore)
    assert.NotNil(t, result.MultiStore)
    assert.Greater(t, result.MultiStore.TotalSavings, 0.0)
}
```

**Integration Tests:**
```go
// tests/integration/stores_test.go
func TestCreateStoreWithDuplicateCheck(t *testing.T) {
    // Setup test DB
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    // Create first store
    store1 := createStore("King Soopers", "123 Main St", "80918")
    
    // Attempt duplicate
    _, err := createStore("King Soopers", "123 Main St", "80918")
    
    // Should fail with duplicate error
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "duplicate")
}
```

**API Tests:**
```bash
# Use httpie or curl for API testing
http POST localhost:8080/api/stores \
  name="Test Store" \
  street_address="123 Test St" \
  city="Colorado Springs" \
  state="CO" \
  zip_code="80918" \
  Authorization:"Bearer $TOKEN"
```

### 4.2 Frontend Testing

> **Vanilla JS Testing:** No Jest, no React Testing Library. We use simple, practical approaches.

**Manual Browser Testing:**
- Use browser DevTools for debugging
- Test in Chrome, Firefox, Safari
- Mobile testing via responsive mode and real devices

**Simple Test Page (for development):**
```html
<!-- web/test.html - Development only, not deployed -->
<!DOCTYPE html>
<html>
<head>
  <title>Frontend Tests</title>
  <script src="/js/api.js"></script>
  <script src="/js/utils.js"></script>
</head>
<body>
  <h1>Frontend Tests</h1>
  <div id="results"></div>

  <script>
    const results = document.getElementById('results');

    async function test(name, fn) {
      try {
        await fn();
        results.innerHTML += `<p style="color:green">✓ ${name}</p>`;
      } catch (err) {
        results.innerHTML += `<p style="color:red">✗ ${name}: ${err.message}</p>`;
      }
    }

    // Run tests
    test('escapeHtml prevents XSS', () => {
      const result = escapeHtml('<script>alert("xss")</script>');
      if (result.includes('<script>')) throw new Error('XSS not escaped');
    });

    test('API client adds auth header', () => {
      localStorage.setItem('token', 'test-token');
      if (api.getToken() !== 'test-token') throw new Error('Token not retrieved');
      localStorage.removeItem('token');
    });
  </script>
</body>
</html>
```

**End-to-End Tests (Playwright - optional):**
```javascript
// e2e/store-creation.spec.js
const { test, expect } = require('@playwright/test');

test('user can create a new store', async ({ page }) => {
  await page.goto('/pages/stores/new.html');

  await page.fill('[name="street_address"]', '123 Test St');
  await page.fill('[name="zip_code"]', '80918');

  // Wait for duplicate check
  await page.waitForSelector('.clear-indicator');

  await page.fill('[name="name"]', 'Test Store');
  await page.selectOption('[name="store_type"]', 'grocery');

  await page.click('button[type="submit"]');

  await expect(page).toHaveURL(/\/pages\/stores\/index\.html/);
});
```

### 4.3 Testing Checklist

**Pre-Launch Testing:**
- [ ] All API endpoints have unit tests (>80% coverage)
- [ ] Database migrations tested (up and down)
- [ ] Duplicate prevention working (stores and items)
- [ ] Price optimization algorithm validated
- [ ] Authentication flow tested
- [ ] Tag autocomplete working
- [ ] Mobile responsive design verified
- [ ] Cross-browser testing (Chrome, Safari, Firefox)
- [ ] Load testing (100 concurrent users)
- [ ] Security audit (SQL injection, XSS, CSRF)

---

## Phase 5: Deployment & DevOps (Week 9-10)

### 5.1 CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run Tests
        run: go test ./... -v

  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to VPS
        uses: appleboy/ssh-action@master
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            cd /opt/price-feed
            git pull origin main

            # Build and restart (single binary serves API + static files)
            go build -o price-feed cmd/server/main.go
            sudo systemctl restart price-feed
```

> **Simplicity:** No npm, no build step for frontend. Just pull, build Go binary, restart. Static files (HTML/CSS/JS) are served directly - no compilation needed.

### 5.2 Systemd Service File

> **Single Service:** Go serves both the API and static files. One binary, one service.

```ini
# /etc/systemd/system/price-feed.service
[Unit]
Description=Price Feed (API + Static Files)
After=postgresql.service network.target

[Service]
Type=simple
User=pricefeed
WorkingDirectory=/opt/price-feed
ExecStart=/opt/price-feed/price-feed
Restart=on-failure
RestartSec=5

# Environment variables
Environment="PORT=8080"
Environment="DATABASE_URL=postgresql://user:pass@localhost/pricefeed"
Environment="JWT_SECRET=your-secret-key"
Environment="STATIC_DIR=./web"

[Install]
WantedBy=multi-user.target
```

### 5.3 Nginx Configuration

> **Single Backend:** Nginx proxies everything to Go on port 8080. Go handles both `/api/*` routes and static files.

```nginx
# /etc/nginx/sites-available/pricefeed
server {
    listen 443 ssl http2;
    server_name pricefeed.foxxcyber.com;

    ssl_certificate /etc/letsencrypt/live/pricefeed.foxxcyber.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pricefeed.foxxcyber.com/privkey.pem;

    # Rate limiting for API
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;

    # API routes
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Static files (HTML, CSS, JS) - served by Go
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Cache static assets
        location ~* \.(css|js|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
            proxy_pass http://localhost:8080;
            expires 7d;
            add_header Cache-Control "public, immutable";
        }
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name pricefeed.foxxcyber.com;
    return 301 https://$host$request_uri;
}
```

### 5.4 Monitoring & Logging

**Logging Strategy:**
```go
// Use structured logging
import "github.com/rs/zerolog/log"

log.Info().
    Str("user_id", userID).
    Str("action", "create_store").
    Msg("Store created successfully")

log.Error().
    Err(err).
    Str("endpoint", "/api/stores").
    Msg("Failed to create store")
```

**Application Monitoring:**
- Setup simple health check endpoint: `GET /health`
- Monitor with UptimeRobot (free tier)
- Setup email alerts for downtime
- Track key metrics:
  - Response times
  - Error rates
  - Active users
  - Database connections

**Log Rotation:**
```bash
# /etc/logrotate.d/price-feed
/var/log/price-feed/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0640 pricefeed pricefeed
    sharedscripts
    postrotate
        systemctl reload price-feed
    endscript
}
```

### 5.5 Backup Strategy

**Database Backups:**
```bash
#!/bin/bash
# /opt/scripts/backup-db.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/pricefeed"
DB_NAME="pricefeed"

pg_dump $DB_NAME | gzip > $BACKUP_DIR/pricefeed_$DATE.sql.gz

# Keep only last 7 days
find $BACKUP_DIR -type f -mtime +7 -delete

# Optional: Upload to S3 or Backblaze B2
```

**Crontab:**
```
# Daily backup at 2 AM
0 2 * * * /opt/scripts/backup-db.sh
```

---

## Phase 6: Beta Testing & Refinement (Week 11-12)

### 6.1 Beta Test Plan

**Participant Recruitment:**
- Target: 20-30 users in Colorado Springs
- Focus: Military spouses, budget-conscious families
- Channels:
  - Local Facebook groups
  - NextDoor posts
  - Military spouse communities
  - Friends and family

**Beta Signup Form:**
- Name, email, zip code
- Primary grocery stores
- Shopping frequency
- Feedback preferences (email, survey, calls)

**Beta Test Objectives:**
- Validate core workflows (create stores, add items, build lists)
- Test duplicate prevention effectiveness
- Gather feedback on UX/UI
- Identify bugs and edge cases
- Measure user engagement metrics

### 6.2 Feedback Collection

**In-App Feedback:**
```html
<!-- Simple feedback widget -->
<button id="feedback-btn" class="feedback-button">Send Feedback</button>

<div id="feedback-modal" class="modal hidden">
  <form id="feedback-form">
    <select name="category" required>
      <option value="">Select category...</option>
      <option value="bug">Bug Report</option>
      <option value="feature">Feature Request</option>
      <option value="general">General Feedback</option>
    </select>
    <textarea name="description" placeholder="Describe your feedback..." required></textarea>
    <input type="file" name="screenshot" accept="image/*">
    <button type="submit">Submit</button>
  </form>
</div>
```

**Weekly Beta Surveys:**
1. **Week 1:** First impressions, onboarding experience
2. **Week 2:** Feature usage, pain points
3. **Week 3:** Price accuracy, usefulness
4. **Week 4:** Overall satisfaction, would you recommend?

**Key Metrics to Track:**
- Daily active users (DAU)
- Stores created per user
- Items added per user
- Price updates per week
- Shopping lists created
- User retention (day 1, day 7, day 30)
- Net Promoter Score (NPS)

### 6.3 Iteration Based on Feedback

**Common Expected Issues:**
- Store address normalization edge cases
- Tag suggestions not matching expectations
- Mobile UI refinements needed
- Performance issues with large lists
- Confusion about verification system

**Quick Win Improvements:**
- Add tooltips/help text where confusion occurs
- Improve error messages
- Add loading states everywhere
- Implement keyboard shortcuts for power users
- Add "Quick Start" tutorial

---

## Phase 7: Public Launch (Week 13-14)

### 7.1 Pre-Launch Checklist

**Technical:**
- [ ] All beta bugs fixed
- [ ] Performance optimized (page load <2s)
- [ ] Mobile responsive verified
- [ ] SEO optimization (meta tags, sitemap)
- [ ] Analytics setup (Google Analytics or Plausible)
- [ ] Error tracking (Sentry or similar)
- [ ] Terms of Service drafted
- [ ] Privacy Policy drafted
- [ ] Contact/Support page created
- [ ] SSL certificates verified
- [ ] Backup system tested
- [ ] Load testing passed (500 users)

**Content:**
- [ ] Landing page copy polished
- [ ] FAQ page created
- [ ] How-to guides written
- [ ] About page with founder story
- [ ] Email templates designed (welcome, verification, etc.)

**Marketing:**
- [ ] Social media accounts created (Facebook, Instagram, Twitter)
- [ ] Launch post drafted
- [ ] Press release prepared (optional)
- [ ] Local news outlets identified
- [ ] Influencer outreach list compiled

### 7.2 Launch Strategy

**Phase 1: Soft Launch (Week 13)**
- Open to Colorado Springs only
- Announce to beta testers first (they seed the database)
- Post in 5-10 local Facebook groups
- Monitor closely for issues
- Target: 100 users in first week

**Phase 2: Local Push (Week 14)**
- Post on NextDoor
- Share in military spouse groups
- Encourage word-of-mouth
- Target: 500 users by end of week

**Phase 3: Regional Expansion (Month 2+)**
- Open to Denver metro
- Add more Colorado cities
- Target: 2,000 users by end of month 2

### 7.3 Growth Tactics

**Viral Mechanics:**
- Referral system: "Invite friends, earn reputation points"
- Social sharing: "I saved $45 this week on groceries with Price Feed!"
- Community challenges: "Help us map all stores in your zip code"

**Content Marketing:**
- Blog posts: "How to Save $100/month on Groceries"
- Local store price comparisons
- Weekly hot deals roundup
- User success stories

**Partnerships:**
- Reach out to local mom blogs
- Partner with community centers
- Sponsor local events

---

## Phase 8: Post-Launch Optimization (Month 2-3)

### 8.1 Feature Additions Based on Usage

**High Priority:**
- Receipt OCR scanning (if users request it)
- Email notifications for price drops
- Mobile-optimized PWA (Progressive Web App)
- Store loyalty card integration
- Meal planning integration
- Budget tracking

**Medium Priority:**
- Browser extension for price capture
- API for third-party integrations
- Advanced filtering and search
- Personalized recommendations
- Price history charts
- Store comparison maps

**Low Priority:**
- Social features (follow users, share lists)
- Gamification (badges, leaderboards)
- Kroger API integration
- Coupon aggregation
- Recipe suggestions

### 8.2 Scaling Considerations

**When to Scale Infrastructure:**
- 5,000+ active users: Upgrade VPS or add second server
- 10,000+ active users: Consider managed PostgreSQL (AWS RDS)
- 50,000+ active users: Implement CDN, caching layer (Redis)
- 100,000+ active users: Migrate to containerized deployment (Docker, Kubernetes)

**Database Optimization:**
- Add read replicas for high traffic
- Implement query caching
- Optimize slow queries (use EXPLAIN ANALYZE)
- Archive old price data (>1 year)
- Partition large tables by region

**Frontend Optimization:**
- Implement lazy loading for images (`loading="lazy"`)
- Optimize images (use WebP format, compress with tools like squoosh)
- Add service worker for PWA (optional)
- Use edge caching (Cloudflare)
- Minify CSS/JS manually or with simple build script (optional)

---

## Cost Breakdown

### Initial Setup (One-Time)
| Item | Cost |
|------|------|
| Domain (pricefeed.foxxcyber.com) | $12/year |
| SSL Certificate | $0 (Let's Encrypt) |
| Development time (your labor) | $0 |
| Logo design (optional) | $50-200 |
| **Total** | **~$100** |

### Monthly Operating Costs (First Year)

**Months 1-3 (<500 users):**
| Item | Cost |
|------|------|
| VPS (8GB RAM) | $40 |
| Cloudflare (free tier) | $0 |
| Email service (SendGrid free) | $0 |
| Monitoring (UptimeRobot free) | $0 |
| **Total** | **$40/month** |

**Months 4-12 (500-5000 users):**
| Item | Cost |
|------|------|
| VPS (16GB RAM) | $80 |
| Cloudflare (free tier) | $0 |
| Email service (SendGrid) | $20 |
| Backups (Backblaze B2) | $5 |
| Error tracking (Sentry) | $0 (free tier) |
| **Total** | **$105/month** |

**Year 2+ (5000-20000 users):**
| Item | Cost |
|------|------|
| VPS or AWS EC2 | $150 |
| Managed DB (AWS RDS) | $50 |
| CDN/Cloudflare Pro | $20 |
| Email service | $50 |
| Monitoring & Analytics | $30 |
| **Total** | **$300/month** |

### Revenue Potential

**Free Model (Ad-supported - optional):**
- 10,000 users × $0.50 CPM × 10 views/day = $50/day = $1,500/month

**Freemium Model (Receipt OCR at $2.99/month):**
- 10,000 users × 5% conversion = 500 paid users
- 500 × $2.99 = $1,495/month

**Break-even:**
- Month 3-4 at current growth trajectory

---

## Risk Mitigation

### Technical Risks

| Risk | Mitigation |
|------|------------|
| Data loss | Daily automated backups, test restoration monthly |
| Server downtime | Monitor 24/7, keep backup VPS config ready |
| Database corruption | Regular integrity checks, replication setup |
| Security breach | Regular security audits, follow OWASP guidelines |
| Scaling issues | Load testing before launch, monitoring in place |

### Business Risks

| Risk | Mitigation |
|------|------------|
| Low user adoption | Start hyper-local, focus on community building |
| Duplicate store spam | Verification system, rate limiting, moderation |
| Inaccurate pricing | Community verification, reputation system |
| Competitive entry | Move fast, build community moat |
| Legal issues (store data) | Consult lawyer, clear ToS, user-generated content disclaimer |

### Operational Risks

| Risk | Mitigation |
|------|------------|
| Time constraints | Focus on MVP, defer non-critical features |
| Burnout | Set realistic timeline, take breaks |
| Scope creep | Stick to implementation plan, phase features |
| Support burden | Comprehensive FAQ, community forum, limit channels |

---

## Success Metrics

### Month 1 Goals
- [ ] 100 registered users
- [ ] 20 stores added and verified
- [ ] 500 items in repository
- [ ] 1,000 price points tracked
- [ ] 50 shopping lists created

### Month 3 Goals
- [ ] 500 registered users
- [ ] 50 stores verified
- [ ] 2,000 items in repository
- [ ] 10,000 price points tracked
- [ ] 30% user retention (30-day)
- [ ] NPS score >40

### Month 6 Goals
- [ ] 2,000 registered users
- [ ] Expand to Denver metro
- [ ] 100+ stores verified
- [ ] 5,000+ items in repository
- [ ] 50,000+ price points tracked
- [ ] 50% user retention (30-day)
- [ ] Break-even on costs

### Year 1 Goals
- [ ] 10,000 registered users
- [ ] Expand to 5+ Colorado cities
- [ ] 500+ stores verified
- [ ] 20,000+ items in repository
- [ ] Self-sustaining (revenue covers costs)
- [ ] PWA fully optimized for mobile
- [ ] Receipt OCR feature launched

---

## Development Timeline

### Detailed Week-by-Week Breakdown

**Week 1: Foundation**
- Mon-Tue: VPS setup, domain, SSL
- Wed-Thu: Database schema, migrations
- Fri-Sun: Go project structure, static file serving setup

**Week 2: Core Backend**
- Mon-Tue: Authentication system (JWT)
- Wed-Thu: Store endpoints with duplicate checking
- Fri-Sun: Item endpoints with tag system

**Week 3: Backend Features**
- Mon-Tue: Price management endpoints
- Wed-Thu: Shopping list endpoints
- Fri-Sun: Feed/activity endpoints, testing

**Week 4: Advanced Backend**
- Mon-Tue: Price optimization algorithm
- Wed-Thu: Verification system
- Fri-Sun: Admin endpoints, comprehensive testing

**Week 5: Frontend Setup**
- Mon-Tue: HTML page structure, CSS base styles
- Wed-Thu: Login/register pages, auth.js
- Fri-Sun: Landing page, navbar component

**Week 6: Frontend Core**
- Mon-Tue: Store management pages (list, add)
- Wed-Thu: Item management pages with tag input
- Fri-Sun: Dashboard, activity feed

**Week 7: Frontend Features**
- Mon-Tue: Shopping list UI
- Wed-Thu: Store comparison/optimization display
- Fri-Sun: Responsive CSS, mobile testing

**Week 8: Polish & Integration**
- Mon-Tue: Wire up all API calls
- Wed-Thu: Error handling, loading states, form validation
- Fri-Sun: Cross-browser testing

**Week 9: Testing & Deployment**
- Mon-Tue: Bug fixes from testing
- Wed-Thu: Deploy to production VPS
- Fri-Sun: CI/CD setup, monitoring

**Week 10: Refinement**
- Mon-Tue: Performance optimization
- Wed-Thu: SEO (meta tags, sitemap), analytics setup
- Fri-Sun: Documentation, Terms/Privacy

**Week 11-12: Beta Testing**
- Recruit testers
- Monitor usage
- Collect feedback
- Fix critical issues
- Iterate on UX

**Week 13-14: Launch**
- Soft launch to Colorado Springs
- Marketing push
- Monitor performance
- Support users
- Gather feedback

---

## Maintenance Plan

### Daily Tasks
- Monitor error logs
- Check uptime status
- Review new user signups
- Moderate flagged content (if any)

### Weekly Tasks
- Review analytics (user growth, engagement)
- Check database performance
- Respond to user feedback
- Update hot deals if applicable
- Backup verification

### Monthly Tasks
- Security updates (OS, dependencies)
- Database optimization
- Review and merge duplicate items/stores
- User engagement analysis
- Feature prioritization review

### Quarterly Tasks
- Infrastructure review (scaling needs?)
- Feature roadmap update
- User survey
- Competitor analysis
- Cost optimization review

---

## Support Strategy

### Communication Channels
- Email: support@pricefeed.foxxcyber.com (primary)
- FAQ page (deflect common questions)
- Community forum (future - Month 3+)

### Response Time Goals
- Critical bugs: <4 hours
- General inquiries: <24 hours
- Feature requests: Acknowledged within 48 hours

### Self-Service Resources
- Comprehensive FAQ
- Video tutorials (optional)
- In-app help tooltips
- "How it works" guide

---

## Legal & Compliance

### Required Documents
1. **Terms of Service**
   - User-generated content
   - Verification system
   - Data accuracy disclaimer
   - Account termination policy

2. **Privacy Policy**
   - Data collection (email, location, usage)
   - Cookie policy
   - Data sharing (none, except as required by law)
   - User rights (access, deletion, export)

3. **Disclaimer**
   - Prices are user-reported
   - No guarantee of accuracy
   - Not responsible for pricing errors
   - Users should verify before purchasing

### Data Privacy (GDPR/CCPA Considerations)
- Users can delete their accounts
- Data export functionality
- Clear consent for data collection
- Minimal data collection (only what's necessary)

---

## Next Steps (Action Items)

### This Week
- [ ] File DBA for "Price Feed" under Foxx Cyber LLC
- [ ] Purchase domain (pricefeed.foxxcyber.com)
- [ ] Provision VPS server
- [ ] Setup development environment (Go, PostgreSQL, VS Code)
- [ ] Create GitHub repository

### Next Week
- [ ] Implement database schema
- [ ] Start Go API development
- [ ] Create base HTML structure and CSS styles
- [ ] Draft Terms of Service and Privacy Policy

### This Month
- [ ] Complete MVP backend (Go API + static file serving)
- [ ] Complete MVP frontend (HTML/CSS/JS pages)
- [ ] Deploy to production
- [ ] Recruit beta testers
- [ ] Launch beta

---

## Conclusion

This implementation plan provides a realistic roadmap from concept to launch in 12-16 weeks. The phased approach allows you to:

1. **Start small** - Focus on Colorado Springs MVP
2. **Validate quickly** - Get real users testing within 2-3 months
3. **Scale smartly** - Grow infrastructure as user base grows
4. **Control costs** - Keep monthly costs under $100 until proven
5. **Iterate rapidly** - Community feedback drives features

The tag-based categorization, address-based duplicate prevention, and community verification systems set this apart from competitors while maintaining data quality.

**Key Success Factors:**
- Ship MVP fast (12 weeks)
- Focus on one city initially
- Build community early
- Listen to user feedback
- Keep infrastructure simple initially
- Scale only when necessary

Ready to start building? Begin with Week 1: Foundation & Infrastructure. Let me know if you need clarification on any section or want me to dive deeper into any specific component!