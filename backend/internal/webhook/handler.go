package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/gitops-drift-detector/internal/drift"
	"go.uber.org/zap"
)

type Handler struct {
	engine *drift.Engine
	log    *zap.Logger
}

func NewHandler(engine *drift.Engine, log *zap.Logger) *Handler {
	return &Handler{engine: engine, log: log}
}

type gitPushEvent struct {
	Repository struct {
		HTMLURL string `json:"html_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	Ref string `json:"ref"`
}

func (h *Handler) HandleGitPush(c *gin.Context) {
	var ev gitPushEvent
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repoURL := ev.Repository.CloneURL
	if repoURL == "" {
		repoURL = ev.Repository.HTMLURL
	}

	h.log.Info("git push webhook received", zap.String("repo", repoURL), zap.String("ref", ev.Ref))

	// TODO: validate HMAC signature from GitHub/GitLab before trusting the payload
	go h.engine.TriggerPollAndDetect(c.Request.Context(), repoURL)

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}
