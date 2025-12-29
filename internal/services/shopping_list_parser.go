package services

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/foxxcyber/price-feed/internal/models"
)

// ShoppingListParser parses Mealie-format shopping lists
type ShoppingListParser struct {
	checkboxPattern *regexp.Regexp
	quantityPattern *regexp.Regexp
	rangePattern    *regexp.Regexp
	fractionPattern *regexp.Regexp
	unitPattern     *regexp.Regexp
}

// Unicode vulgar fractions mapping
var unicodeFractions = map[rune]float64{
	'\u00BC': 0.25,     // ¼
	'\u00BD': 0.5,      // ½
	'\u00BE': 0.75,     // ¾
	'\u2150': 0.142857, // ⅐
	'\u2151': 0.111111, // ⅑
	'\u2152': 0.1,      // ⅒
	'\u2153': 0.333333, // ⅓
	'\u2154': 0.666667, // ⅔
	'\u2155': 0.2,      // ⅕
	'\u2156': 0.4,      // ⅖
	'\u2157': 0.6,      // ⅗
	'\u2158': 0.8,      // ⅘
	'\u2159': 0.166667, // ⅙
	'\u215A': 0.833333, // ⅚
	'\u215B': 0.125,    // ⅛
	'\u215C': 0.375,    // ⅜
	'\u215D': 0.625,    // ⅝
	'\u215E': 0.875,    // ⅞
}

// Superscript digits for fractions like ¹/₂
var superscriptDigits = map[rune]int{
	'\u2070': 0, '\u00B9': 1, '\u00B2': 2, '\u00B3': 3,
	'\u2074': 4, '\u2075': 5, '\u2076': 6, '\u2077': 7,
	'\u2078': 8, '\u2079': 9,
}

// Subscript digits for fractions like ¹/₂
var subscriptDigits = map[rune]int{
	'\u2080': 0, '\u2081': 1, '\u2082': 2, '\u2083': 3,
	'\u2084': 4, '\u2085': 5, '\u2086': 6, '\u2087': 7,
	'\u2088': 8, '\u2089': 9,
}

// Unit normalization map
var unitNormalization = map[string]string{
	// Volume - small
	"tsp":          "teaspoon",
	"t":            "teaspoon",
	"teaspoons":    "teaspoon",
	"tbsp":         "tablespoon",
	"tbs":          "tablespoon",
	"tablespoons":  "tablespoon",
	"fl oz":        "fluid ounce",
	"floz":         "fluid ounce",
	"fluid ounces": "fluid ounce",

	// Volume - medium
	"c":      "cup",
	"cups":   "cup",
	"pt":     "pint",
	"pints":  "pint",
	"qt":     "quart",
	"quarts": "quart",

	// Volume - large
	"gal":         "gallon",
	"gallons":     "gallon",
	"l":           "liter",
	"liters":      "liter",
	"litres":      "liter",
	"ml":          "milliliter",
	"milliliters": "milliliter",

	// Weight
	"oz":        "ounce",
	"ounces":    "ounce",
	"lb":        "pound",
	"lbs":       "pound",
	"pounds":    "pound",
	"g":         "gram",
	"grams":     "gram",
	"kg":        "kilogram",
	"kilograms": "kilogram",

	// Count
	"pc":       "piece",
	"pcs":      "piece",
	"pieces":   "piece",
	"ct":       "count",
	"ea":       "each",
	"pk":       "pack",
	"pkg":      "package",
	"packages": "package",
	"bunch":    "bunch",
	"bunches":  "bunch",
	"head":     "head",
	"heads":    "head",
	"clove":    "clove",
	"cloves":   "clove",
	"sprig":    "sprig",
	"sprigs":   "sprig",
	"stalk":    "stalk",
	"stalks":   "stalk",
	"slice":    "slice",
	"slices":   "slice",
	"can":      "can",
	"cans":     "can",
	"jar":      "jar",
	"jars":     "jar",
	"bag":      "bag",
	"bags":     "bag",
	"box":      "box",
	"boxes":    "box",
	"bottle":   "bottle",
	"bottles":  "bottle",
	"stick":    "stick",
	"sticks":   "stick",
	"dash":     "dash",
	"dashes":   "dash",
	"pinch":    "pinch",
	"pinches":  "pinch",
}

// NewShoppingListParser creates a new parser instance
func NewShoppingListParser() *ShoppingListParser {
	return &ShoppingListParser{
		// Match markdown checkbox lines: - [ ] or - [x]
		checkboxPattern: regexp.MustCompile(`^\s*-\s*\[[ xX]?\]\s*(.+)$`),

		// Match quantity at start: 1, 1.5, etc.
		quantityPattern: regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*`),

		// Match quantity range: 2.5 - 3
		rangePattern: regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*-\s*(\d+(?:\.\d+)?)\s*`),

		// Match ASCII fraction: 1/2, 3/4
		fractionPattern: regexp.MustCompile(`^(\d+)/(\d+)\s*`),

		// Match units (case insensitive) - order matters, longer patterns first
		unitPattern: regexp.MustCompile(`(?i)^(tablespoons?|teaspoons?|fluid ounces?|milliliters?|kilograms?|packages?|gallons?|bottles?|bunches?|ounces?|pounds?|pieces?|liters?|sprigs?|stalks?|slices?|cloves?|quarts?|pinch(?:es)?|pints?|dashes?|sticks?|heads?|grams?|boxes?|cups?|cans?|jars?|bags?|tbsp|floz|tsp|tbs|pkg|gal|cup|qt|pt|oz|lb|ml|kg|ct|ea|pk|pc|g|l|c)\b\s*`),
	}
}

// Parse parses markdown content and returns structured items
func (p *ShoppingListParser) Parse(content string) ([]models.ParsedShoppingItem, error) {
	lines := strings.Split(content, "\n")
	var items []models.ParsedShoppingItem
	lineNumber := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract content from checkbox line
		matches := p.checkboxPattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue // Skip non-checkbox lines
		}

		content := strings.TrimSpace(matches[1])
		item := p.parseLine(content, line, lineNumber)
		items = append(items, item)
		lineNumber++
	}

	return items, nil
}

// parseLine parses a single line into structured data
func (p *ShoppingListParser) parseLine(content, rawText string, lineNumber int) models.ParsedShoppingItem {
	item := models.ParsedShoppingItem{
		RawText:    rawText,
		LineNumber: lineNumber,
		Quantity:   1.0, // Default quantity
	}

	remaining := content

	// Step 1: Extract quantity (including fractions and ranges)
	remaining, item.Quantity = p.extractQuantity(remaining)

	// Step 2: Extract unit
	remaining, item.Unit = p.extractUnit(remaining)

	// Step 3: Extract notes (content in parentheses or after comma)
	remaining, item.Notes = p.extractNotes(remaining)

	// Step 4: Clean up name
	item.Name = p.cleanName(remaining)

	return item
}

// extractQuantity handles all quantity formats
func (p *ShoppingListParser) extractQuantity(s string) (string, float64) {
	s = strings.TrimSpace(s)
	quantity := 1.0

	// Check for range first (e.g., "2.5 - 3")
	if matches := p.rangePattern.FindStringSubmatch(s); len(matches) == 3 {
		low, _ := strconv.ParseFloat(matches[1], 64)
		high, _ := strconv.ParseFloat(matches[2], 64)
		quantity = (low + high) / 2 // Use average
		s = strings.TrimSpace(s[len(matches[0]):])
		return s, quantity
	}

	// Check for whole number + Unicode fraction (e.g., "1 ½")
	wholeAndUnicodeFraction := regexp.MustCompile(`^(\d+)\s+`)
	if matches := wholeAndUnicodeFraction.FindStringSubmatch(s); len(matches) == 2 {
		afterWhole := strings.TrimSpace(s[len(matches[0]):])
		rest, unicodeQty := p.extractUnicodeFraction(afterWhole)
		if unicodeQty > 0 {
			whole, _ := strconv.ParseFloat(matches[1], 64)
			return rest, whole + unicodeQty
		}
	}

	// Check for whole number + ASCII fraction (e.g., "1 1/2")
	wholeAndFractionPattern := regexp.MustCompile(`^(\d+)\s+(\d+)/(\d+)\s*`)
	if matches := wholeAndFractionPattern.FindStringSubmatch(s); len(matches) == 4 {
		whole, _ := strconv.ParseFloat(matches[1], 64)
		num, _ := strconv.ParseFloat(matches[2], 64)
		denom, _ := strconv.ParseFloat(matches[3], 64)
		if denom != 0 {
			quantity = whole + (num / denom)
		}
		s = strings.TrimSpace(s[len(matches[0]):])
		return s, quantity
	}

	// Check for Unicode fractions and superscript/subscript combinations at the start
	rest, unicodeQty := p.extractUnicodeFraction(s)
	if unicodeQty > 0 {
		return rest, unicodeQty
	}

	// Check for simple fraction (e.g., "1/2")
	if matches := p.fractionPattern.FindStringSubmatch(s); len(matches) == 3 {
		num, _ := strconv.ParseFloat(matches[1], 64)
		denom, _ := strconv.ParseFloat(matches[2], 64)
		if denom != 0 {
			quantity = num / denom
		}
		s = strings.TrimSpace(s[len(matches[0]):])
		return s, quantity
	}

	// Check for decimal or whole number (e.g., "1.5", "2")
	if matches := p.quantityPattern.FindStringSubmatch(s); len(matches) == 2 {
		qty, _ := strconv.ParseFloat(matches[1], 64)
		quantity = qty
		s = strings.TrimSpace(s[len(matches[0]):])
	}

	return s, quantity
}

// extractUnicodeFraction handles Unicode vulgar fractions and superscript/subscript
func (p *ShoppingListParser) extractUnicodeFraction(s string) (string, float64) {
	if len(s) == 0 {
		return s, 0
	}

	runes := []rune(s)
	idx := 0

	// Skip leading whitespace
	for idx < len(runes) && unicode.IsSpace(runes[idx]) {
		idx++
	}

	if idx >= len(runes) {
		return s, 0
	}

	// Check for single Unicode vulgar fraction
	if val, ok := unicodeFractions[runes[idx]]; ok {
		remaining := string(runes[idx+1:])
		return strings.TrimSpace(remaining), val
	}

	// Check for superscript/subscript fraction (e.g., ¹/₂ or ¹⁄₂)
	numStart := idx
	var numerator int = 0
	hasNumerator := false
	for idx < len(runes) {
		if digit, ok := superscriptDigits[runes[idx]]; ok {
			numerator = numerator*10 + digit
			idx++
			hasNumerator = true
		} else {
			break
		}
	}

	// Check for fraction slash (U+2044) or regular slash after superscript
	if hasNumerator && idx < len(runes) && (runes[idx] == '\u2044' || runes[idx] == '/') {
		idx++

		// Look for subscript denominator
		var denominator int = 0
		hasDenominator := false
		for idx < len(runes) {
			if digit, ok := subscriptDigits[runes[idx]]; ok {
				denominator = denominator*10 + digit
				idx++
				hasDenominator = true
			} else {
				break
			}
		}

		if hasDenominator && denominator > 0 {
			result := float64(numerator) / float64(denominator)
			remaining := string(runes[idx:])
			return strings.TrimSpace(remaining), result
		}
	}

	// Reset if we didn't find a valid fraction
	_ = numStart

	return s, 0
}

// extractUnit extracts and normalizes the unit
func (p *ShoppingListParser) extractUnit(s string) (string, string) {
	s = strings.TrimSpace(s)

	if matches := p.unitPattern.FindStringSubmatch(s); len(matches) >= 2 {
		unit := strings.ToLower(matches[1])
		if normalized, ok := unitNormalization[unit]; ok {
			unit = normalized
		}
		remaining := strings.TrimSpace(s[len(matches[0]):])
		return remaining, unit
	}

	return s, ""
}

// extractNotes extracts content in parentheses or after comma
func (p *ShoppingListParser) extractNotes(s string) (string, string) {
	var notes []string

	// Extract parenthetical content
	parenPattern := regexp.MustCompile(`\(([^)]+)\)`)
	if matches := parenPattern.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		for _, m := range matches {
			notes = append(notes, strings.TrimSpace(m[1]))
		}
		s = parenPattern.ReplaceAllString(s, "")
	}

	// Extract content after comma
	if idx := strings.Index(s, ","); idx >= 0 {
		afterComma := strings.TrimSpace(s[idx+1:])
		if afterComma != "" {
			notes = append(notes, afterComma)
		}
		s = s[:idx]
	}

	return strings.TrimSpace(s), strings.Join(notes, "; ")
}

// cleanName cleans up the item name
func (p *ShoppingListParser) cleanName(s string) string {
	s = strings.TrimSpace(s)

	// Remove trailing punctuation
	s = strings.TrimRight(s, ".,;:-_")

	// Collapse multiple spaces
	spacePattern := regexp.MustCompile(`\s+`)
	s = spacePattern.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}
