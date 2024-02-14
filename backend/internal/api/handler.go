package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/gitops-drift-detector/internal/alert"
	"github.com/yourorg/gitops-drift-detector/internal/audit"
	"github.com/yourorg/gitops-drift-detector/internal/cluster"
	"github.com/yourorg/gitops-drift-detector/internal/drift"
	"github.com/yourorg/gitops-drift-detector/internal/git"
	"github.com/yourorg/gitops-drift-detector/internal/reconcile"
	"go.uber.org/zap"
)

type Handler struct {
	clusters   *cluster.Manager
	git        *git.Poller
	engine     *drift.Engine
	reconciler *reconcile.Reconciler
	audit      *audit.Logger
	alerter    *alert.Alerter
	log        *zap.Logger
}

func NewHandler(
	clusters *cluster.Manager,
	git *git.Poller,
	engine *drift.Engine,
	reconciler *reconcile.Reconciler,
	audit *audit.Logger,
	alerter *alert.Alerter,
	log *zap.Logger,
) *Handler {
	return &Handler{
		clusters:   clusters,
		git:        git,
		engine:     engine,
		reconciler: reconciler,
		audit:      audit,
		alerter:    alerter,
		log:        log,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	v1.GET("/clusters", h.listClusters)
	v1.POST("/clusters", h.addCluster)
	v1.DELETE("/clusters/:id", h.removeCluster)

	v1.GET("/clusters/:id/drift", h.getDrift)
	v1.POST("/clusters/:id/drift/trigger", h.triggerDetection)

	v1.POST("/clusters/:id/reconcile", h.reconcile)

	v1.GET("/clusters/:id/audit", h.getAudit)

	v1.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
}

func (h *Handler) listClusters(c *gin.Context) {
	c.JSON(http.StatusOK, h.clusters.List())
}

type addClusterReq struct {
	Name       string `json:"name" binding:"required"`
	Kubeconfig string `json:"kubeconfig"`
}

func (h *Handler) addCluster(c *gin.Context) {
	var req addClusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// generate a stable ID from name for now
	// TODO: persist clusters to the DB so they survive restarts
	id := req.Name
	if err := h.clusters.AddCluster(id, req.Name, req.Kubeconfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) removeCluster(c *gin.Context) {
	h.clusters.RemoveCluster(c.Param("id"))
	c.Status(http.StatusNoContent)
}

func (h *Handler) getDrift(c *gin.Context) {
	events := h.engine.GetLatest(c.Param("id"))
	if events == nil {
		events = []drift.DriftEvent{}
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) triggerDetection(c *gin.Context) {
	go h.engine.TriggerDetection(c.Request.Context(), c.Param("id"))
	c.JSON(http.StatusAccepted, gin.H{"status": "triggered"})
}

func (h *Handler) reconcile(c *gin.Context) {
	var req reconcile.ReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ClusterID = c.Param("id")
	req.Actor = c.GetHeader("X-Actor")
	if req.Actor == "" {
		req.Actor = "api"
	}

	if err := h.reconciler.Apply(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reconciled"})
}

func (h *Handler) getAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	entries, err := h.audit.Query(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}
