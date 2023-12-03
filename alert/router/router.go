package router

import (
	"net/http"

	"cncamp/pkg/third_party/nightingale/alert/aconf"
	"cncamp/pkg/third_party/nightingale/alert/astats"
	"cncamp/pkg/third_party/nightingale/alert/process"
	"cncamp/pkg/third_party/nightingale/memsto"
	"cncamp/pkg/third_party/nightingale/pkg/ctx"
	"cncamp/pkg/third_party/nightingale/pkg/httpx"
	"github.com/gin-gonic/gin"
)

type Router struct {
	HTTP               httpx.Config
	Alert              aconf.Alert
	AlertMuteCache     *memsto.AlertMuteCacheType
	TargetCache        *memsto.TargetCacheType
	BusiGroupCache     *memsto.BusiGroupCacheType
	AlertStats         *astats.Stats
	Ctx                *ctx.Context
	ExternalProcessors *process.ExternalProcessorsType
}

func New(httpConfig httpx.Config, alert aconf.Alert, amc *memsto.AlertMuteCacheType, tc *memsto.TargetCacheType, bgc *memsto.BusiGroupCacheType,
	astats *astats.Stats, ctx *ctx.Context, externalProcessors *process.ExternalProcessorsType) *Router {
	return &Router{
		HTTP:               httpConfig,
		Alert:              alert,
		AlertMuteCache:     amc,
		TargetCache:        tc,
		BusiGroupCache:     bgc,
		AlertStats:         astats,
		Ctx:                ctx,
		ExternalProcessors: externalProcessors,
	}
}

func (rt *Router) Config(r *gin.Engine) {
	if !rt.HTTP.Alert.Enable {
		return
	}

	service := r.Group("/v1/n9e")
	if len(rt.HTTP.Alert.BasicAuth) > 0 {
		service.Use(gin.BasicAuth(rt.HTTP.Alert.BasicAuth))
	}
	service.POST("/event", rt.pushEventToQueue)
	service.POST("/make-event", rt.makeEvent)
}

func Render(c *gin.Context, data, msg interface{}) {
	if msg == nil {
		if data == nil {
			data = struct{}{}
		}
		c.JSON(http.StatusOK, gin.H{"data": data, "error": ""})
	} else {
		c.JSON(http.StatusOK, gin.H{"error": gin.H{"message": msg}})
	}
}

func Dangerous(c *gin.Context, v interface{}, code ...int) {
	if v == nil {
		return
	}

	switch t := v.(type) {
	case string:
		if t != "" {
			c.JSON(http.StatusOK, gin.H{"error": gin.H{"message": v}})
		}
	case error:
		c.JSON(http.StatusOK, gin.H{"error": gin.H{"message": t.Error()}})
	}
}
