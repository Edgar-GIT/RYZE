package exercise_media

import (
	"strings"

	"ryze/backend/config"
)

// Media is the safe media projection of one exercise. All fields are empty and
// Available is false whenever media exposure is disabled.
type Media struct {
	VideoURL     string
	ImageURL     string
	AnimationURL string
	Attribution  string
	Available    bool
}

// Resolver applies the server-wide exercise-media policy to stored media
// references. Stored values are relative paths; the configured base path is
// prefixed only when exposure is enabled.
type Resolver struct {
	enabled  bool
	basePath string
}

func New(cfg config.ExerciseMediaConfig) Resolver {
	return Resolver{enabled: cfg.Enabled, basePath: cfg.BasePath}
}

// Resolve gates every media reference behind the server configuration. When a
// reference is enabled the stored relative paths are prefixed with the
// configured same-origin base path. Attribution is returned only when at least
// one usable reference exists.
func (r Resolver) Resolve(videoURL, imageURL, animationURL, attribution string) Media {
	if !r.enabled {
		return Media{}
	}
	media := Media{
		VideoURL:     join(r.basePath, videoURL),
		ImageURL:     join(r.basePath, imageURL),
		AnimationURL: join(r.basePath, animationURL),
		Attribution:  attribution,
	}
	media.Available = media.VideoURL != "" || media.ImageURL != "" || media.AnimationURL != ""
	if !media.Available {
		return Media{}
	}
	return media
}

func join(basePath, mediaPath string) string {
	if mediaPath == "" {
		return ""
	}
	return strings.TrimSuffix(basePath, "/") + "/" + strings.TrimPrefix(mediaPath, "/")
}
