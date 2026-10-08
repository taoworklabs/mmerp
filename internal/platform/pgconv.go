package platform

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

// OrErr returns err, or fallback when err is nil; for "failed, or refused".
func OrErr(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}
