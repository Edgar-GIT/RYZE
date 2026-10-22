package exercise_media_test

import (
	"testing"

	"ryze/backend/config"
	"ryze/backend/services/exercise_media"
)

func TestResolveDisabledClearsEveryMediaField(t *testing.T) {
	resolver := exercise_media.New(config.ExerciseMediaConfig{Enabled: false, BasePath: "/media/exercises"})

	media := resolver.Resolve("videos/demo.mp4", "images/still.jpg", "videos/demo.gif", "© Gym visual")

	if media != (exercise_media.Media{}) {
		t.Fatalf("expected empty media while disabled, got %+v", media)
	}
}

func TestResolveEnabledPrefixesPathsAndKeepsAttribution(t *testing.T) {
	resolver := exercise_media.New(config.ExerciseMediaConfig{Enabled: true, BasePath: "/media/exercises/"})

	media := resolver.Resolve("videos/demo.mp4", "/images/still.jpg", "videos/demo.gif", "© Gym visual")

	if media.VideoURL != "/media/exercises/videos/demo.mp4" {
		t.Fatalf("unexpected video URL: %q", media.VideoURL)
	}
	if media.ImageURL != "/media/exercises/images/still.jpg" {
		t.Fatalf("unexpected image URL: %q", media.ImageURL)
	}
	if media.AnimationURL != "/media/exercises/videos/demo.gif" {
		t.Fatalf("unexpected animation URL: %q", media.AnimationURL)
	}
	if media.Attribution != "© Gym visual" || !media.Available {
		t.Fatalf("unexpected attribution/availability: %+v", media)
	}
}

func TestResolveEnabledWithoutReferencesIsUnavailable(t *testing.T) {
	resolver := exercise_media.New(config.ExerciseMediaConfig{Enabled: true, BasePath: "/media/exercises"})

	media := resolver.Resolve("", "", "", "unused attribution")

	if media != (exercise_media.Media{}) {
		t.Fatalf("expected empty media without references, got %+v", media)
	}
}
