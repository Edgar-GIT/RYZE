package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ryze/backend/models"
	"ryze/backend/services/program_details"
	uploads "ryze/backend/services/public_uploads"
)

// UploadsHandler stores programme cover images and serves the public uploads.
// The destination folder is always derived on the server from the plan's
// catalogue category; the client never names a path.
type UploadsHandler struct {
	store *uploads.Store
}

func NewUploadsHandler(store *uploads.Store) *UploadsHandler {
	return &UploadsHandler{store: store}
}

type uploadedImageResponse struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// AdminProgramCover stores a cover image for a platform plan. The form field
// "family" (generic, premium_level_1 or premium_level_2) selects the category
// folder.
func (h *UploadsHandler) AdminProgramCover(c *gin.Context) {
	family := strings.TrimSpace(c.PostForm("family"))
	switch family {
	case models.ProgramProductTypeGeneric, models.ProgramProductTypePremiumLevel1, models.ProgramProductTypePremiumLevel2:
	default:
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	h.saveCover(c, program_details.CoverFolder(family, models.FamilyProgramType(family)))
}

// TrainerProgramCover stores a cover image for a trainer plan. The form field
// "type" (free, premium or personalized) selects the category folder.
func (h *UploadsHandler) TrainerProgramCover(c *gin.Context) {
	programType := strings.TrimSpace(c.PostForm("type"))
	switch programType {
	case models.ProgramTypeFree, models.ProgramTypePremium, models.ProgramTypePersonalized:
	default:
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	h.saveCover(c, program_details.CoverFolder(models.ProgramProductTypeGeneric, programType))
}

func (h *UploadsHandler) saveCover(c *gin.Context, folder string) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	defer file.Close()

	key, err := h.store.SaveProgramCover(c.Request.Context(), folder, file)
	if err != nil {
		switch {
		case errors.Is(err, uploads.ErrUnsupportedType), errors.Is(err, uploads.ErrInvalidImage), errors.Is(err, uploads.ErrEmpty):
			RespondError(c, http.StatusBadRequest, "IMAGE_TYPE_NOT_ALLOWED", "Only JPEG, PNG and WebP images are accepted.", nil)
		case errors.Is(err, uploads.ErrTooLarge):
			RespondError(c, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "The image exceeds the maximum allowed size.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	keyCopy := key
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Image uploaded successfully.",
		"data":    uploadedImageResponse{Key: key, URL: program_details.CoverURL(&keyCopy)},
	})
}

// ServeProgramImage serves one stored programme image. Only keys of the exact
// store-generated shape are resolved, so no other file under the uploads
// directory (or outside it) can ever be reached. Keys are unique per upload,
// which makes the response safe to cache for a long time.
func (h *UploadsHandler) ServeProgramImage(c *gin.Context) {
	key := "programs" + c.Param("filepath")
	if !uploads.ServedKeyPattern.MatchString(key) || !h.store.Exists(key) {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", uploads.ContentTypeFor(key))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(h.store.Path(key))
}
