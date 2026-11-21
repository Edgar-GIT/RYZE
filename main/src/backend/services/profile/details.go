package profile

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ryze/backend/repositories"
)

// The public "about" of a profile (owner request, 2026-11-21): biography,
// age (from the date of birth), years of training (from the year they
// started), weight and social networks. Everything is optional.

const MaxBioLength = 500

// InvalidDetailsError lists the fields that failed validation.
type InvalidDetailsError struct{ Fields []string }

func (e *InvalidDetailsError) Error() string {
	return "invalid profile details: " + strings.Join(e.Fields, ", ")
}

// SocialPlatform describes how a network's handle is read and linked.
type SocialPlatform struct {
	Key     string
	handle  *regexp.Regexp
	hosts   []string
	path    *regexp.Regexp // extracts the handle from a link path
	linkFor func(handle string) string
}

// SocialPlatforms are the networks a profile can link to, in display order.
var SocialPlatforms = []SocialPlatform{
	{Key: "instagram", handle: regexp.MustCompile(`^[A-Za-z0-9._]{1,30}$`), hosts: []string{"instagram.com", "www.instagram.com"},
		path: regexp.MustCompile(`^/([A-Za-z0-9._]{1,30})/?$`), linkFor: func(h string) string { return "https://www.instagram.com/" + h + "/" }},
	{Key: "tiktok", handle: regexp.MustCompile(`^[A-Za-z0-9._]{2,24}$`), hosts: []string{"tiktok.com", "www.tiktok.com"},
		path: regexp.MustCompile(`^/@([A-Za-z0-9._]{2,24})/?$`), linkFor: func(h string) string { return "https://www.tiktok.com/@" + h }},
	{Key: "youtube", handle: regexp.MustCompile(`^[A-Za-z0-9._-]{3,30}$`), hosts: []string{"youtube.com", "www.youtube.com", "m.youtube.com"},
		path: regexp.MustCompile(`^/@([A-Za-z0-9._-]{3,30})/?$`), linkFor: func(h string) string { return "https://www.youtube.com/@" + h }},
	{Key: "x", handle: regexp.MustCompile(`^[A-Za-z0-9_]{1,15}$`), hosts: []string{"x.com", "www.x.com", "twitter.com", "www.twitter.com"},
		path: regexp.MustCompile(`^/([A-Za-z0-9_]{1,15})/?$`), linkFor: func(h string) string { return "https://x.com/" + h }},
	{Key: "facebook", handle: regexp.MustCompile(`^[A-Za-z0-9.]{5,50}$`), hosts: []string{"facebook.com", "www.facebook.com", "m.facebook.com"},
		path: regexp.MustCompile(`^/([A-Za-z0-9.]{5,50})/?$`), linkFor: func(h string) string { return "https://www.facebook.com/" + h }},
	{Key: "strava", handle: regexp.MustCompile(`^[0-9]{1,12}$`), hosts: []string{"strava.com", "www.strava.com"},
		path: regexp.MustCompile(`^/athletes/([0-9]{1,12})/?$`), linkFor: func(h string) string { return "https://www.strava.com/athletes/" + h }},
}

func platformByKey(key string) (*SocialPlatform, bool) {
	for i := range SocialPlatforms {
		if SocialPlatforms[i].Key == key {
			return &SocialPlatforms[i], true
		}
	}
	return nil, false
}

// SocialLink is one network on a profile.
type SocialLink struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle"`
	URL      string `json:"url"`
}

// LinksFor returns the stored handles as links, in display order.
func LinksFor(handles map[string]string) []SocialLink {
	links := []SocialLink{}
	for _, platform := range SocialPlatforms {
		if handle := handles[platform.Key]; handle != "" && platform.handle.MatchString(handle) {
			links = append(links, SocialLink{Platform: platform.Key, Handle: handle, URL: platform.linkFor(handle)})
		}
	}
	return links
}

// NormalizeSocialHandle accepts "@handle", "handle" or a link to the profile
// on the platform's own site and returns the handle.
func NormalizeSocialHandle(platformKey, value string) (string, bool) {
	platform, ok := platformByKey(platformKey)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, "/") {
		if !strings.Contains(value, "://") {
			value = "https://" + value
		}
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return "", false
		}
		host := strings.ToLower(parsed.Hostname())
		allowed := false
		for _, candidate := range platform.hosts {
			if host == candidate {
				allowed = true
			}
		}
		match := platform.path.FindStringSubmatch(parsed.Path)
		if !allowed || match == nil {
			return "", false
		}
		value = match[1]
	}
	value = strings.TrimPrefix(value, "@")
	if !platform.handle.MatchString(value) {
		return "", false
	}
	return value, true
}

// DetailsInput is what the person sends; empty strings clear a field.
type DetailsInput struct {
	Bio           string            `json:"bio"`
	BirthDate     string            `json:"birth_date"`
	TrainingSince *int              `json:"training_since"`
	WeightKg      *float64          `json:"weight_kg"`
	Socials       map[string]string `json:"socials"`
}

// ProfileDetails is the stored "about" as the owner edits it.
type ProfileDetails struct {
	Bio           string       `json:"bio"`
	BirthDate     string       `json:"birth_date"`
	TrainingSince *int         `json:"training_since"`
	WeightKg      *float64     `json:"weight_kg"`
	Socials       []SocialLink `json:"socials"`
}

// DetailsRepository stores the details.
type DetailsRepository interface {
	Get(ctx context.Context, userID string) (*repositories.ProfileDetailsRow, error)
	Save(ctx context.Context, userID string, details repositories.ProfileDetailsRow) error
}

// DetailsService reads and saves the owner's details.
type DetailsService struct {
	repo DetailsRepository
	now  func() time.Time
}

func NewDetailsService(repo DetailsRepository) *DetailsService {
	return &DetailsService{repo: repo, now: time.Now}
}

func toProfileDetails(row *repositories.ProfileDetailsRow) *ProfileDetails {
	details := &ProfileDetails{TrainingSince: row.TrainingSince, WeightKg: row.WeightKg, Socials: LinksFor(row.SocialLinks)}
	if row.Bio != nil {
		details.Bio = *row.Bio
	}
	if row.BirthDate != nil {
		details.BirthDate = row.BirthDate.Format("2006-01-02")
	}
	return details
}

func (s *DetailsService) Get(ctx context.Context, userID string) (*ProfileDetails, error) {
	row, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toProfileDetails(row), nil
}

// Save validates every field: a biography up to 500 characters (line breaks
// allowed), an age from 13 to 100, a training start year from 1950 to this
// year, a weight from 20 to 400 kg and handles in each network's format.
func (s *DetailsService) Save(ctx context.Context, userID string, input DetailsInput) (*ProfileDetails, error) {
	var invalid []string
	row := repositories.ProfileDetailsRow{SocialLinks: map[string]string{}}
	now := s.now().UTC()

	bio := strings.TrimSpace(input.Bio)
	if utf8.RuneCountInString(bio) > MaxBioLength || strings.ContainsFunc(bio, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) {
		invalid = append(invalid, "bio")
	} else if bio != "" {
		row.Bio = &bio
	}

	if birth := strings.TrimSpace(input.BirthDate); birth != "" {
		day, err := time.Parse("2006-01-02", birth)
		if age := ageOn(day, now); err != nil || age < 13 || age > 100 {
			invalid = append(invalid, "birth_date")
		} else {
			row.BirthDate = &day
		}
	}

	if input.TrainingSince != nil {
		if *input.TrainingSince < 1950 || *input.TrainingSince > now.Year() {
			invalid = append(invalid, "training_since")
		} else {
			row.TrainingSince = input.TrainingSince
		}
	}

	if input.WeightKg != nil {
		weight := math.Round(*input.WeightKg*10) / 10
		if math.IsNaN(weight) || weight < 20 || weight > 400 {
			invalid = append(invalid, "weight_kg")
		} else {
			row.WeightKg = &weight
		}
	}

	for key, value := range input.Socials {
		if strings.TrimSpace(value) == "" {
			continue
		}
		handle, ok := NormalizeSocialHandle(key, value)
		if !ok {
			invalid = append(invalid, "socials."+key)
			continue
		}
		row.SocialLinks[key] = handle
	}

	if len(invalid) > 0 {
		return nil, &InvalidDetailsError{Fields: invalid}
	}
	if err := s.repo.Save(ctx, userID, row); err != nil {
		return nil, fmt.Errorf("save profile details: %w", err)
	}
	return toProfileDetails(&row), nil
}

func ageOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

// PublicAbout is what the public profile shows of the details.
type PublicAbout struct {
	Bio           string       `json:"bio"`
	Age           *int         `json:"age"`
	TrainingYears *int         `json:"training_years"`
	WeightKg      *float64     `json:"weight_kg"`
	Socials       []SocialLink `json:"socials"`
}

// Public returns the public "about" of an account.
func (s *DetailsService) Public(ctx context.Context, userID string) (*PublicAbout, error) {
	row, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	about := &PublicAbout{WeightKg: row.WeightKg, Socials: LinksFor(row.SocialLinks)}
	if row.Bio != nil {
		about.Bio = *row.Bio
	}
	if row.BirthDate != nil {
		age := ageOn(*row.BirthDate, now)
		about.Age = &age
	}
	if row.TrainingSince != nil {
		years := now.Year() - *row.TrainingSince
		about.TrainingYears = &years
	}
	return about, nil
}

// IsInvalidDetails reports a validation error and its fields.
func IsInvalidDetails(err error) ([]string, bool) {
	var invalid *InvalidDetailsError
	if errors.As(err, &invalid) {
		return invalid.Fields, true
	}
	return nil, false
}
