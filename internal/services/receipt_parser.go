package services

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/foxxcyber/price-feed/internal/models"
)

// ReceiptParser parses OCR text from receipts
type ReceiptParser struct {
	pricePatterns   []*regexp.Regexp
	excludePatterns []*regexp.Regexp
	datePatterns    []*regexp.Regexp
	totalPatterns   []*regexp.Regexp
}

// NewReceiptParser creates a new receipt parser
func NewReceiptParser() *ReceiptParser {
	return &ReceiptParser{
		pricePatterns: []*regexp.Regexp{
			// Pattern: Commissary format - ITEM NAME UPC $X.XX F (UPC is 11-14 digits)
			// Examples: CANDY PNUT BTR 00034000004409 $1.18 F, MILK WHOLE GALL 00015700146019 $3.02 F
			regexp.MustCompile(`^(.+?)\s+\d{11,14}\s+\$?(\d{1,3}\.\d{2})\s*[FNT]?\s*$`),
			// Pattern: ITEM NAME    $X.XX or ITEM NAME    X.XX (price at end)
			regexp.MustCompile(`^(.+?)\s+\$?(\d{1,3}\.\d{2})\s*$`),
			// Pattern: ITEM NAME @ X.XX EA
			regexp.MustCompile(`^(.+?)\s+@\s*\$?(\d{1,3}\.\d{2})\s*(?:EA|EACH)?`),
			// Pattern: QTY x ITEM @ PRICE or QTY ITEM @ PRICE
			regexp.MustCompile(`^(\d+)\s*[xX@]\s*(.+?)\s+\$?(\d{1,3}\.\d{2})`),
			// Pattern: ITEM    PRICE F (with tax flag)
			regexp.MustCompile(`^(.+?)\s+\$?(\d{1,3}\.\d{2})\s*[FNT]?\s*$`),
		},
		excludePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^\s*(TAX|SUBTOTAL|SUB\s*TOTAL|TOTAL|GRAND\s*TOTAL|BALANCE|CHANGE|CASH|CREDIT|DEBIT|CARD|VISA|MASTERCARD|AMEX|DISCOVER|SAVINGS|DISCOUNT|COUPON|MEMBER|LOYALTY|POINTS|REWARD|THANK\s*YOU|HAVE\s*A|STORE\s*#|CASHIER|TRANS|REG|DATE|TIME|TEL|PHONE|ADDRESS|RECEIPT|RETURN|REFUND|VOID|SURCHARGE|SOLD\s*ITEMS?|PAID|PURCHASE|CREDIT\s*CARD)\b`),
			regexp.MustCompile(`^\s*[-=*]+\s*$`),
			regexp.MustCompile(`^\s*\d{2}[/-]\d{2}[/-]\d{2,4}\s*$`),
			regexp.MustCompile(`^\s*\d{1,2}:\d{2}\s*(AM|PM)?\s*$`),
			// Commissary category headers
			regexp.MustCompile(`(?i)^\s*(BREAD\s*(AND|&)\s*SNACKS|DAIRY|PACKAGE\s*FOOD|PRE\s*PACKAGED\s*MEAT|PRODUCE|SPECIALTY\s*FOODS?|FROZEN\s*FOODS?|BEVERAGES?|DELI|BAKERY|MEAT|SEAFOOD|GROCERY|HEALTH\s*(AND|&)\s*BEAUTY|HOUSEHOLD|PET\s*SUPPLIES?)\s*$`),
			// Quantity/weight detail lines: "2 @ $2.79 EACH" or "2.96 lb @ $0.99 / lb"
			regexp.MustCompile(`^\s*\d+\.?\d*\s*(lb|oz|kg|g)?\s*@\s*\$?\d+\.\d{2}\s*(\/\s*(lb|oz|kg|g)|EACH|EA)?\s*$`),
		},
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(\d{1,2})[/-](\d{1,2})[/-](\d{2,4})`),
			regexp.MustCompile(`(\d{4})[/-](\d{1,2})[/-](\d{1,2})`),
		},
		totalPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:TOTAL|GRAND\s*TOTAL|BALANCE\s*DUE|AMOUNT\s*DUE)\s*:?\s*\$?(\d+\.\d{2})`),
			regexp.MustCompile(`(?i)^\s*TOTAL\s+\$?(\d+\.\d{2})`),
		},
	}
}

// Parse parses OCR text and extracts receipt data
func (p *ReceiptParser) Parse(ocrText string) (*models.ParsedReceipt, error) {
	lines := strings.Split(ocrText, "\n")
	result := &models.ParsedReceipt{
		Items: []models.ParsedItem{},
	}

	// Extract date
	result.Date = p.extractDate(lines)

	// Extract total
	result.Total = p.extractTotal(lines)

	// Parse item lines
	lineNumber := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip excluded lines
		if p.shouldExclude(line) {
			continue
		}

		// Try to parse as a price line
		item := p.parseLine(line, lineNumber)
		if item != nil {
			result.Items = append(result.Items, *item)
			lineNumber++
		}
	}

	return result, nil
}

// parseLine attempts to parse a line as an item with price
func (p *ReceiptParser) parseLine(line string, lineNumber int) *models.ParsedItem {
	// Clean up the line
	line = p.cleanLine(line)

	for _, pattern := range p.pricePatterns {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) >= 3 {
			var name string
			var priceStr string
			var quantity int = 1

			if len(matches) == 4 {
				// Pattern with quantity: QTY, NAME, PRICE
				qty, err := strconv.Atoi(matches[1])
				if err == nil {
					quantity = qty
				}
				name = matches[2]
				priceStr = matches[3]
			} else {
				// Pattern without quantity: NAME, PRICE
				name = matches[1]
				priceStr = matches[2]
			}

			// Parse price
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				continue
			}

			// Clean name
			name = p.cleanItemName(name)
			if name == "" {
				continue
			}

			// Skip if price is unreasonable (likely a phone number or other number)
			if price <= 0 || price > 9999 {
				continue
			}

			return &models.ParsedItem{
				RawText:    line,
				Name:       name,
				Price:      price,
				Quantity:   quantity,
				LineNumber: lineNumber,
			}
		}
	}

	return nil
}

// shouldExclude checks if a line should be excluded
func (p *ReceiptParser) shouldExclude(line string) bool {
	for _, pattern := range p.excludePatterns {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// cleanLine cleans up a line for parsing
func (p *ReceiptParser) cleanLine(line string) string {
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`\s+`)
	line = spaceRe.ReplaceAllString(line, " ")

	// Remove common OCR artifacts
	line = strings.ReplaceAll(line, "|", "")
	line = strings.ReplaceAll(line, "\\", "")

	return strings.TrimSpace(line)
}

// cleanItemName cleans up an item name
func (p *ReceiptParser) cleanItemName(name string) string {
	name = strings.TrimSpace(name)

	// Remove trailing punctuation
	name = strings.TrimRight(name, ".,;:-_")

	// Remove common prefixes
	prefixes := []string{"@", "#", "*"}
	for _, prefix := range prefixes {
		name = strings.TrimPrefix(name, prefix)
	}

	return strings.TrimSpace(name)
}

// extractDate extracts a date from the receipt
func (p *ReceiptParser) extractDate(lines []string) *time.Time {
	for _, line := range lines {
		for _, pattern := range p.datePatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) >= 4 {
				var year, month, day int
				var err error

				// Try MM/DD/YYYY or MM-DD-YYYY format
				month, err = strconv.Atoi(matches[1])
				if err != nil {
					continue
				}
				day, err = strconv.Atoi(matches[2])
				if err != nil {
					continue
				}
				year, err = strconv.Atoi(matches[3])
				if err != nil {
					continue
				}

				// Handle 2-digit years
				if year < 100 {
					if year > 50 {
						year += 1900
					} else {
						year += 2000
					}
				}

				// Check if it's actually YYYY-MM-DD format
				if matches[1] != "" && len(matches[1]) == 4 {
					year, _ = strconv.Atoi(matches[1])
					month, _ = strconv.Atoi(matches[2])
					day, _ = strconv.Atoi(matches[3])
				}

				// Validate date
				if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
					date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
					return &date
				}
			}
		}
	}
	return nil
}

// extractTotal extracts the total from the receipt
func (p *ReceiptParser) extractTotal(lines []string) *float64 {
	// Search from the bottom of the receipt
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		for _, pattern := range p.totalPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) >= 2 {
				total, err := strconv.ParseFloat(matches[1], 64)
				if err == nil && total > 0 {
					return &total
				}
			}
		}
	}
	return nil
}
