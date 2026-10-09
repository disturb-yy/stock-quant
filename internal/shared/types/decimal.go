package types

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var decimalPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// Decimal is a normalized, exact base-ten value safe for JSON and SQL text boundaries.
type Decimal struct {
	value string
}

// ParseDecimal parses and normalizes a JSON-compatible decimal representation.
func ParseDecimal(value string) (Decimal, error) {
	if len(value) == 0 || len(value) > 256 || !decimalPattern.MatchString(value) {
		return Decimal{}, errors.New("decimal must be a finite base-ten number")
	}
	coefficient, scale, err := decimalParts(value)
	if err != nil || len(coefficient.String()) > 256 {
		return Decimal{}, errors.New("decimal is outside the supported range")
	}
	return Decimal{value: formatDecimal(coefficient, scale)}, nil
}

// DecimalFromFloat64 explicitly converts a finite binary floating-point value to its shortest decimal text.
func DecimalFromFloat64(value float64) (Decimal, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Decimal{}, errors.New("decimal float64 must be finite")
	}
	return ParseDecimal(strconv.FormatFloat(value, 'g', -1, 64))
}

// String returns the normalized decimal text, or an explicit invalid marker for the zero value.
func (value Decimal) String() string {
	if !value.Valid() {
		return "<invalid-decimal>"
	}
	return value.value
}

// Valid reports whether this Decimal has a valid value.
func (value Decimal) Valid() bool {
	return value.value != "" && decimalPattern.MatchString(value.value)
}

// Fits reports whether a value fits DECIMAL(precision, scale) without rounding.
func (value Decimal) Fits(precision, scale int) bool {
	coefficient, decimalScale, err := decimalParts(value.value)
	if err != nil || precision <= 0 || scale < 0 || scale > precision || decimalScale > scale {
		return false
	}
	digits := len(strings.TrimPrefix(strings.TrimPrefix(coefficient.String(), "-"), "+"))
	if digits == 0 {
		digits = 1
	}
	integerDigits := digits - decimalScale
	if coefficient.Sign() == 0 {
		integerDigits = 0
	}
	if integerDigits < 0 {
		integerDigits = 0
	}
	return integerDigits <= precision-scale
}

// MultiplyInt64 multiplies a decimal exactly by an integer factor.
func (value Decimal) MultiplyInt64(factor int64) (Decimal, error) {
	coefficient, scale, err := decimalParts(value.value)
	if err != nil {
		return Decimal{}, errors.New("cannot multiply an invalid decimal")
	}
	coefficient.Mul(coefficient, big.NewInt(factor))
	result, err := ParseDecimal(formatDecimal(coefficient, scale))
	if err != nil {
		return Decimal{}, fmt.Errorf("multiply decimal: %w", err)
	}
	return result, nil
}

// Cmp compares exact decimal values.
func (value Decimal) Cmp(other Decimal) (int, error) {
	left, err := decimalRat(value.value)
	if err != nil {
		return 0, errors.New("compare invalid decimal")
	}
	right, err := decimalRat(other.value)
	if err != nil {
		return 0, errors.New("compare invalid decimal")
	}
	return left.Cmp(right), nil
}

// Float64Checked explicitly converts to float64 and rejects overflow, invalid values, and nonzero underflow.
func (value Decimal) Float64Checked() (float64, error) {
	if !value.Valid() {
		return 0, errors.New("cannot convert invalid decimal to float64")
	}
	parsed, err := strconv.ParseFloat(value.value, 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || parsed == 0 && value.value != "0" {
		return 0, errors.New("decimal is not representable as finite float64")
	}
	return parsed, nil
}

// MarshalJSON emits the exact decimal as an unquoted JSON number.
func (value Decimal) MarshalJSON() ([]byte, error) {
	if !value.Valid() {
		return nil, errors.New("cannot marshal invalid decimal")
	}
	return []byte(value.value), nil
}

// UnmarshalJSON parses an exact JSON number without converting through float64.
func (value *Decimal) UnmarshalJSON(data []byte) error {
	if value == nil {
		return errors.New("cannot unmarshal decimal into nil receiver")
	}
	parsed, err := ParseDecimal(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

// AmountYuan stores an exact monetary value measured in Chinese yuan.
type AmountYuan struct {
	value Decimal
}

// NewAmountYuan explicitly converts a finite float64 into an exact decimal representation.
func NewAmountYuan(value float64) (AmountYuan, error) {
	decimal, err := DecimalFromFloat64(value)
	if err != nil {
		return AmountYuan{}, fmt.Errorf("amount in yuan: %w", err)
	}
	return NewAmountYuanDecimal(decimal)
}

// NewAmountYuanDecimal constructs an exact amount without sign or scale restrictions.
func NewAmountYuanDecimal(value Decimal) (AmountYuan, error) {
	if !value.Valid() {
		return AmountYuan{}, errors.New("amount in yuan must be a valid decimal")
	}
	return AmountYuan{value: value}, nil
}

// Decimal returns the exact amount in yuan.
func (amount AmountYuan) Decimal() Decimal { return amount.value }

// String returns the normalized amount in yuan.
func (amount AmountYuan) String() string { return amount.value.String() }

// Float64Checked explicitly converts the amount to a finite float64.
func (amount AmountYuan) Float64Checked() (float64, error) { return amount.value.Float64Checked() }

func decimalParts(value string) (*big.Int, int, error) {
	exponent := 0
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		parsed, err := strconv.Atoi(value[index+1:])
		if err != nil || parsed > 256 || parsed < -256 {
			return nil, 0, errors.New("decimal exponent is out of range")
		}
		exponent = parsed
		value = value[:index]
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	whole, fraction, hasFraction := strings.Cut(value, ".")
	coefficient, ok := new(big.Int).SetString(whole+fraction, 10)
	if !ok {
		return nil, 0, errors.New("decimal coefficient is invalid")
	}
	if negative {
		coefficient.Neg(coefficient)
	}
	scale := 0
	if hasFraction {
		scale = len(fraction)
	}
	scale -= exponent
	if scale < 0 {
		coefficient.Mul(coefficient, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scale)), nil))
		scale = 0
	}
	for scale > 0 && new(big.Int).Mod(new(big.Int).Abs(new(big.Int).Set(coefficient)), big.NewInt(10)).Sign() == 0 {
		coefficient.Quo(coefficient, big.NewInt(10))
		scale--
	}
	return coefficient, scale, nil
}

func formatDecimal(coefficient *big.Int, scale int) string {
	negative := coefficient.Sign() < 0
	digits := new(big.Int).Abs(new(big.Int).Set(coefficient)).String()
	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		point := len(digits) - scale
		digits = digits[:point] + "." + digits[point:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}

func decimalRat(value string) (*big.Rat, error) {
	coefficient, scale, err := decimalParts(value)
	if err != nil {
		return nil, err
	}
	numerator := new(big.Int).Set(coefficient)
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	return new(big.Rat).SetFrac(numerator, denominator), nil
}
