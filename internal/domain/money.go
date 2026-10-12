package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const defaultCurrency = "BRL"

var scientificNotation = regexp.MustCompile(`(?i)[eE]`)

type Money struct {
	Amount   int64  `json:"-"`
	Currency string `json:"currency"`
}

func NewMoney(amount int64, currency string) (Money, error) {
	if currency == "" {
		currency = defaultCurrency
	}
	if amount < 0 {
		return Money{}, fmt.Errorf("money amount cannot be negative")
	}
	return Money{Amount: amount, Currency: strings.ToUpper(currency)}, nil
}

func ParseMoneyString(raw string) (Money, error) {
	if raw == "" {
		return Money{}, fmt.Errorf("money amount is required")
	}
	if scientificNotation.MatchString(raw) {
		return Money{}, fmt.Errorf("scientific notation not allowed")
	}
	if strings.Contains(raw, "NaN") || strings.Contains(raw, "Inf") || strings.Contains(raw, "Infinity") {
		return Money{}, fmt.Errorf("invalid numeric value")
	}
	if strings.HasPrefix(raw, "+") {
		raw = strings.TrimPrefix(raw, "+")
	}
	if strings.Contains(raw, ".") {
		parts := strings.Split(raw, ".")
		if len(parts) != 2 {
			return Money{}, fmt.Errorf("invalid decimal format")
		}
		if len(parts[1]) > 2 {
			return Money{}, fmt.Errorf("money amount can only have two decimal places")
		}
		whole := parts[0]
		if whole == "" {
			whole = "0"
		}
		frac := parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if whole == "-" {
			return Money{}, fmt.Errorf("invalid money amount")
		}
		if strings.HasPrefix(whole, "-") {
			return Money{}, fmt.Errorf("financial external inputs cannot be negative")
		}
		v, err := strconv.ParseInt(whole, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("invalid integer part: %w", err)
		}
		f, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("invalid decimal part: %w", err)
		}
		minor := v*100 + f
		if minor < 0 {
			return Money{}, fmt.Errorf("overflow while parsing money")
		}
		return Money{Amount: minor, Currency: defaultCurrency}, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid integer money amount: %w", err)
	}
	if v < 0 {
		return Money{}, fmt.Errorf("financial external inputs cannot be negative")
	}
	return Money{Amount: v * 100, Currency: defaultCurrency}, nil
}

func (m Money) IsZero() bool { return m.Amount == 0 }

func (m Money) String() string {
	if m.Amount == 0 {
		return "0.00"
	}
	negative := m.Amount < 0
	abs := absInt64(m.Amount)
	whole := abs / 100
	frac := abs % 100
	if negative {
		return fmt.Sprintf("-%d.%02d", whole, frac)
	}
	return fmt.Sprintf("%d.%02d", whole, frac)
}

func (m Money) Add(other Money) (Money, error) {
	if !m.SameCurrency(other) {
		return Money{}, fmt.Errorf("currency mismatch")
	}
	result := m.Amount + other.Amount
	if result < math.MinInt64 || result > math.MaxInt64 {
		return Money{}, fmt.Errorf("money overflow")
	}
	return Money{Amount: result, Currency: m.Currency}, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if !m.SameCurrency(other) {
		return Money{}, fmt.Errorf("currency mismatch")
	}
	result := m.Amount - other.Amount
	if result < math.MinInt64 || result > math.MaxInt64 {
		return Money{}, fmt.Errorf("money overflow")
	}
	return Money{Amount: result, Currency: m.Currency}, nil
}

func (m Money) Negate() (Money, error) {
	if m.Amount == math.MinInt64 {
		return Money{}, fmt.Errorf("money overflow")
	}
	return Money{Amount: -m.Amount, Currency: m.Currency}, nil
}

func (m Money) SameCurrency(other Money) bool {
	return strings.EqualFold(m.Currency, other.Currency)
}

func (m Money) GreaterThanOrEqual(other Money) bool {
	if !m.SameCurrency(other) {
		return false
	}
	return m.Amount >= other.Amount
}

func (m Money) MarshalJSON() ([]byte, error) {
	type moneyJSON struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	return json.Marshal(moneyJSON{Amount: m.String(), Currency: strings.ToUpper(m.Currency)})
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var payload struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	parsed, err := ParseMoneyString(payload.Amount)
	if err != nil {
		return err
	}
	if payload.Currency != "" {
		parsed.Currency = strings.ToUpper(payload.Currency)
	}
	*m = parsed
	return nil
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func NewZeroMoney(currency string) Money {
	if currency == "" {
		currency = defaultCurrency
	}
	return Money{Amount: 0, Currency: strings.ToUpper(currency)}
}
