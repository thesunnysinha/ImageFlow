package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thesunnysinha/imageflow/apps/api/internal/jobs"
)

type handlers struct{ d Deps }

type createJobRequest struct {
	WebhookURL string   `json:"webhook_url"`
	SourceURLs []string `json:"source_urls" binding:"required,min=1"`
}

func (h *handlers) createJob(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if len(req.SourceURLs) > h.d.MaxItemsPerJob {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "too many source_urls"})
		return
	}
	ctx := c.Request.Context()
	for i, u := range req.SourceURLs {
		if err := h.d.ValidateURL(ctx, u); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid source_url", "index": i})
			return
		}
	}
	var webhook *string
	if req.WebhookURL != "" {
		if err := h.d.ValidateURL(ctx, req.WebhookURL); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid webhook_url"})
			return
		}
		webhook = &req.WebhookURL
	}
	job, err := h.d.Store.Create(ctx, c.GetString(ownerKey), webhook, req.SourceURLs)
	if err != nil {
		h.d.Logger.Error("create job", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Header("Location", "/v1/jobs/"+job.ID)
	c.JSON(http.StatusAccepted, job)
}

func (h *handlers) getJob(c *gin.Context) {
	job, err := h.d.Store.Get(c.Request.Context(), c.GetString(ownerKey), c.Param("id"))
	if errors.Is(err, jobs.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		h.d.Logger.Error("get job", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, job)
}
