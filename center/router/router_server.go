package router

import (
	"cncamp/pkg/third_party/nightingale/models"
	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

func (rt *Router) serversGet(c *gin.Context) {
	list, err := models.AlertingEngineGets(rt.Ctx, "")
	ginx.NewRender(c).Data(list, err)
}

func (rt *Router) serverClustersGet(c *gin.Context) {
	list, err := models.AlertingEngineGetsClusters(rt.Ctx, "")
	ginx.NewRender(c).Data(list, err)
}
