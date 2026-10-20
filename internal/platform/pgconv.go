package platform

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// Conversions between pgtype nullables and the pointers and strings the API uses.

func TextPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func NullText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

func Int8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func NullInt4(p *int32) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *p, Valid: true}
}

func NullInt8(p *int64) pgtype.Int8 {
	if p == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *p, Valid: true}
}

// DatePtr writes a date as YYYY-MM-DD.
func DatePtr(v pgtype.Date) *string {
	if !v.Valid {
		return nil
	}
	s := v.Time.Format(time.DateOnly)
	return &s
}

// NullDate parses a YYYY-MM-DD date the API already validated.
func NullDate(p *string) pgtype.Date {
	if p == nil {
		return pgtype.Date{}
	}
	t, err := time.Parse(time.DateOnly, *p)
	return pgtype.Date{Time: t, Valid: err == nil}
}

// PlainDecimal writes a numeric column's text without trailing zeros: "1.0" is "1", "2.5" stays.
// Text that is not a number comes back as is.
func PlainDecimal(s string) string {
	if d, err := decimal.NewFromString(s); err == nil {
		return d.String()
	}
	return s
}

// Instant writes a timestamptz as RFC 3339 in UTC.
func Instant(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	return new(t.Time.UTC().Format(time.RFC3339))
}

// MonthPeriod turns YYYY-MM into its first and last day; a bad month is a 422 on field "month".
func MonthPeriod(month string) (start, end pgtype.Date, err error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return start, end, &Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"month"}}}
	}
	return pgtype.Date{Time: t, Valid: true}, pgtype.Date{Time: t.AddDate(0, 1, -1), Valid: true}, nil
}

// Optional is nil for an empty string.
func Optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// OrErr returns err, or fallback when err is nil; for "failed, or refused".
func OrErr(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}
