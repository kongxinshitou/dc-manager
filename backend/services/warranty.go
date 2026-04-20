package services

import (
	"time"
)

const (
	WarrantyIn    = "in_warranty"
	WarrantyOut   = "out_of_warranty"
	WarrantyNone  = "unknown"
)

// CalcWarrantyStatus determines warranty status dynamically.
// Priority: if warrantyYears > 0 and warrantyStart is set, compute end = start + years;
// otherwise fall back to warrantyEnd vs today.
func CalcWarrantyStatus(warrantyStart, warrantyEnd *time.Time, warrantyYears int) string {
	if warrantyStart != nil && warrantyYears > 0 {
		endDate := warrantyStart.AddDate(warrantyYears, 0, 0)
		if time.Now().Before(endDate) {
			return WarrantyIn
		}
		return WarrantyOut
	}
	if warrantyEnd != nil {
		if time.Now().Before(*warrantyEnd) {
			return WarrantyIn
		}
		return WarrantyOut
	}
	return WarrantyNone
}

// WarrantyExpiringDays returns the number of days until warranty expires.
// Returns -1 if already expired or no warranty info.
func WarrantyExpiringDays(warrantyStart, warrantyEnd *time.Time, warrantyYears int) int {
	var endDate time.Time
	if warrantyStart != nil && warrantyYears > 0 {
		endDate = warrantyStart.AddDate(warrantyYears, 0, 0)
	} else if warrantyEnd != nil {
		endDate = *warrantyEnd
	} else {
		return -1
	}
	days := int(time.Until(endDate).Hours() / 24)
	if days < 0 {
		return -1
	}
	return days
}
