package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/repositories"
)

// ProgramArchiveHandler archives a published programme (off the market for a
// while) and restores it (owner request, 2026-11-27). Trainers act on their own
// programmes; the Management Administrator on the platform's, per family.
type ProgramArchiveHandler struct {
	repo repositories.ProgramArchiveRepository
	// family is the platform catalogue family for the admin routes ("" for the
	// trainer routes).
	family string
}

func NewProgramArchiveHandler(repo repositories.ProgramArchiveRepository, family string) *ProgramArchiveHandler {
	return &ProgramArchiveHandler{repo: repo, family: family}
}

func (h *ProgramArchiveHandler) owner(c *gin.Context) (repositories.ProgramOwner, bool) {
	if h.family != "" {
		return repositories.ProgramOwner{ProductType: h.family}, true
	}
	trainerID, ok := requireTrainerContext(c)
	return repositories.ProgramOwner{TrainerID: trainerID}, ok
}

func (h *ProgramArchiveHandler) Archive(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	program, err := h.repo.Archive(c.Request.Context(), owner, c.Param("programID"), time.Now().UTC())
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Programme archived.", "data": gin.H{"id": program.ID, "archived_at": program.ArchivedAt}})
}

func (h *ProgramArchiveHandler) Restore(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	program, err := h.repo.Restore(c.Request.Context(), owner, c.Param("programID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Programme back on the market.", "data": gin.H{"id": program.ID, "archived_at": nil}})
}

// Count reports how many programmes the owner keeps archived and the limit.
func (h *ProgramArchiveHandler) Count(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	count, err := h.repo.CountArchived(c.Request.Context(), owner.TrainerID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Archived programmes.", "data": gin.H{"archived": count, "limit": repositories.MaxArchivedPrograms}})
}

func (h *ProgramArchiveHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Programme not found.", nil)
	case errors.Is(err, repositories.ErrArchiveLimit):
		RespondError(c, http.StatusConflict, "ARCHIVE_LIMIT_REACHED", "You can keep at most 15 archived programmes. Restore or delete one first.", nil)
	case errors.Is(err, repositories.ErrArchiveState):
		RespondError(c, http.StatusConflict, "PROGRAM_NOT_ARCHIVABLE", "Only a published programme can be archived, and only an archived one restored.", nil)
	default:
		slog.Error("programme archive failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
