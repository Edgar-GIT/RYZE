package public_uploads

import (
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"regexp"
)

// Workout photos (owner request, 2026-11-22, as in Lyfta): a member can add a
// photo of themselves to a workout. The picture is re-encoded as a JPEG (no
// original metadata) with its longest side at most SessionPhotoMaxSide. The
// folder is private: no public route serves it; the owner reads it through an
// authenticated endpoint only.
const SessionPhotoMaxSide = 1280

// SessionPhotoKeyPattern is the only shape a workout photo key can have.
var SessionPhotoKeyPattern = regexp.MustCompile(`^session_photos/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jpg$`)

// SaveSessionPhoto stores one workout photo and returns its key.
func (s *Store) SaveSessionPhoto(ctx context.Context, content io.Reader) (string, error) {
	return s.saveFitted(ctx, "session_photos", content)
}

// saveFitted stores a picture re-encoded as a JPEG, longest side at most
// SessionPhotoMaxSide, in folder.
func (s *Store) saveFitted(ctx context.Context, folder string, content io.Reader) (string, error) {
	source, err := s.decodeUpload(ctx, content)
	if err != nil {
		return "", err
	}
	return s.storeJPEG(folder, fitWithin(source, SessionPhotoMaxSide))
}

// Feedback images (owner request, 2026-11-23): one optional picture per
// member feedback, in the "feedback" folder. No public uploads route serves
// the folder: the application serves an image only once its feedback is
// approved (and to the reviewers).
var FeedbackImageKeyPattern = regexp.MustCompile(`^feedback/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jpg$`)

// SaveFeedbackImage stores one feedback image and returns its key.
func (s *Store) SaveFeedbackImage(ctx context.Context, content io.Reader) (string, error) {
	return s.saveFitted(ctx, "feedback", content)
}

// DeleteFeedbackImage removes a stored feedback image; unknown keys are ignored.
func (s *Store) DeleteFeedbackImage(key string) error {
	if !FeedbackImageKeyPattern.MatchString(key) {
		return nil
	}
	if err := os.Remove(s.Path(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// FeedbackImagePath returns the file of a valid, existing feedback image key.
func (s *Store) FeedbackImagePath(key string) (string, bool) {
	if !FeedbackImageKeyPattern.MatchString(key) {
		return "", false
	}
	path := s.Path(key)
	info, err := os.Stat(path)
	return path, err == nil && info.Mode().IsRegular()
}

// DeleteSessionPhoto removes a stored workout photo; unknown keys are ignored.
func (s *Store) DeleteSessionPhoto(key string) error {
	if !SessionPhotoKeyPattern.MatchString(key) {
		return nil
	}
	if err := os.Remove(s.Path(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// SessionPhotoPath returns the file of a valid, existing workout photo key.
func (s *Store) SessionPhotoPath(key string) (string, bool) {
	if !SessionPhotoKeyPattern.MatchString(key) {
		return "", false
	}
	path := s.Path(key)
	info, err := os.Stat(path)
	return path, err == nil && info.Mode().IsRegular()
}

// fitWithin scales src down (never up) so its longest side is at most
// maxSide, averaging each destination pixel's source area.
func fitWithin(src image.Image, maxSide int) *image.RGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	outW, outH := width, height
	if width >= height && width > maxSide {
		outW, outH = maxSide, height*maxSide/width
	} else if height > width && height > maxSide {
		outW, outH = width*maxSide/height, maxSide
	}
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < outH; y++ {
		sy0 := bounds.Min.Y + y*height/outH
		sy1 := bounds.Min.Y + (y+1)*height/outH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < outW; x++ {
			sx0 := bounds.Min.X + x*width/outW
			sx1 := bounds.Min.X + (x+1)*width/outW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r += uint64(cr) + uint64(0xffff-ca)
					g += uint64(cg) + uint64(0xffff-ca)
					b += uint64(cb) + uint64(0xffff-ca)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n >> 8), G: uint8(g / n >> 8), B: uint8(b / n >> 8), A: 0xff})
		}
	}
	return dst
}
