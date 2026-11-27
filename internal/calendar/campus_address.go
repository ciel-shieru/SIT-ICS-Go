package calendar

import "strings"

// SITCampusAddress is the public SIT campus address appended to the LOCATION
// property of in-campus events (ADR-0024). The exact byte layout is part of
// the contract: leading newline, trailing space after the institution name,
// newline, then the street address.
const SITCampusAddress = "\nSingapore Institute of Technology \n1 Punggol Coast Rd, Singapore 828608"

// sitVenuePlaceholders are undetermined venue values that must not receive
// the campus address: smart merge (internal/smartmerge) matches exactly this
// set plus "online" as potential Zoom online meetings (ADR-0024).
var sitVenuePlaceholders = map[string]struct{}{
	"tbd":           {},
	"tba":           {},
	"to be advised": {},
}

// isBrightSpaceSource reports whether source belongs to a BrightSpace/Xsite
// event (ADR-0013).
func isBrightSpaceSource(source string) bool {
	return strings.HasPrefix(source, "brightspace-")
}

// IsConcreteVenue reports whether location names a concrete on-campus venue:
// non-empty after trimming, not "online" (case-insensitive, trimmed), and
// not an undetermined placeholder (ADR-0024).
func IsConcreteVenue(location string) bool {
	loc := strings.ToLower(strings.TrimSpace(location))
	if loc == "" || loc == "online" {
		return false
	}
	if _, ok := sitVenuePlaceholders[loc]; ok {
		return false
	}
	return true
}

// WithSITCampusAddress returns location with SITCampusAddress appended when
// the event is in-campus (non-BrightSpace source and concrete venue),
// otherwise location unchanged. The original location is trimmed before
// appending. It is idempotent: a location already ending in
// SITCampusAddress is returned unchanged (ADR-0024).
func WithSITCampusAddress(source, location string) string {
	if isBrightSpaceSource(source) {
		return location
	}
	if !IsConcreteVenue(location) {
		return location
	}
	if strings.HasSuffix(location, SITCampusAddress) {
		return location
	}
	return strings.TrimSpace(location) + SITCampusAddress
}
