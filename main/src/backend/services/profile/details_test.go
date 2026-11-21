package profile

import (
	"context"
	"testing"
	"time"

	"ryze/backend/repositories"
)

func TestNormalizeSocialHandle(t *testing.T) {
	cases := []struct {
		platform, value, want string
		ok                    bool
	}{
		{"instagram", "@rita.lifts", "rita.lifts", true},
		{"instagram", "https://www.instagram.com/rita.lifts/", "rita.lifts", true},
		{"instagram", "instagram.com/rita.lifts", "rita.lifts", true},
		{"instagram", "https://evil.example/rita", "", false},
		{"tiktok", "https://www.tiktok.com/@rita.lifts", "rita.lifts", true},
		{"youtube", "https://youtube.com/@RitaLifts", "RitaLifts", true},
		{"x", "https://twitter.com/rita_l", "rita_l", true},
		{"strava", "https://www.strava.com/athletes/12345", "12345", true},
		{"strava", "rita", "", false},
		{"myspace", "rita", "", false},
		{"instagram", "javascript:alert(1)", "", false},
	}
	for _, tc := range cases {
		got, ok := NormalizeSocialHandle(tc.platform, tc.value)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s %q = %q %v, want %q %v", tc.platform, tc.value, got, ok, tc.want, tc.ok)
		}
	}
}

type memoryDetails struct {
	row repositories.ProfileDetailsRow
}

func (m *memoryDetails) Get(context.Context, string) (*repositories.ProfileDetailsRow, error) {
	row := m.row
	if row.SocialLinks == nil {
		row.SocialLinks = map[string]string{}
	}
	return &row, nil
}

func (m *memoryDetails) Save(_ context.Context, _ string, row repositories.ProfileDetailsRow) error {
	m.row = row
	return nil
}

func TestDetailsSaveValidatesAndPublishes(t *testing.T) {
	repo := &memoryDetails{}
	service := NewDetailsService(repo)
	service.now = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	since, weight, early := 2020, 78.44, 1900
	if _, err := service.Save(ctx, "u", DetailsInput{BirthDate: "2020-01-01", TrainingSince: &early, Socials: map[string]string{"instagram": "https://evil.example/x"}}); err == nil {
		t.Fatalf("expected a validation error")
	} else if fields, ok := IsInvalidDetails(err); !ok || len(fields) != 3 {
		t.Fatalf("invalid fields = %v", fields)
	}

	saved, err := service.Save(ctx, "u", DetailsInput{
		Bio: "  Powerlifter.\\nCoach.  ", BirthDate: "1996-11-22", TrainingSince: &since, WeightKg: &weight,
		Socials: map[string]string{"instagram": "@rita.lifts", "x": ""},
	})
	if err != nil || saved.WeightKg == nil || *saved.WeightKg != 78.4 || len(saved.Socials) != 1 {
		t.Fatalf("save = %+v, %v", saved, err)
	}
	about, err := service.Public(ctx, "u")
	if err != nil || about.Age == nil || *about.Age != 29 || about.TrainingYears == nil || *about.TrainingYears != 6 {
		t.Fatalf("public about = %+v, %v", about, err)
	}
	if about.Socials[0].URL != "https://www.instagram.com/rita.lifts/" {
		t.Fatalf("link = %q", about.Socials[0].URL)
	}
}
