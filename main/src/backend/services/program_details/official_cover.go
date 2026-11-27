package program_details

import (
	_ "embed"
	"strings"
)

// An official RYZE plan (owner request, 2026-11-23) is a free plan that
// ADMIN_2 marks as the platform's own instead of a custom one. Its only
// difference is the cover, which nobody can choose or replace: on wide
// (horizontal) views a colour background with the word RYZE, and on portrait
// cards the RYZE "Base plan" picture. Both ship with the API (no upload
// needed). The cover key serves the wide picture, so every place that shows
// a plan cover shows it without special handling; the portrait plan cards ask
// for OfficialPortraitKey.
const (
	OfficialCoverKey    = "programs/official/ryze_base_plan.png"
	OfficialPortraitKey = "programs/official/ryze_base_plan_portrait.png"
)

var (
	//go:embed assets/ryze_base_plan_wide.png
	officialWide []byte
	//go:embed assets/ryze_base_plan_portrait.png
	officialPortrait []byte
)

// OfficialCoverImage returns the official picture served at key (PNG), or nil
// when key is not one of them.
func OfficialCoverImage(key string) []byte {
	switch key {
	case OfficialCoverKey:
		return officialWide
	case OfficialPortraitKey:
		return officialPortrait
	}
	return nil
}

// IsOfficialCover reports whether a plan's cover marks it as an official
// RYZE plan.
func IsOfficialCover(key *string) bool {
	return key != nil && strings.TrimSpace(*key) == OfficialCoverKey
}
