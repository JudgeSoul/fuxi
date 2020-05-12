package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/apps/v1"
	"net/http"
)

// Get DaemonSet
func (w *WorkloadsAPI) GetDaemonSet(g *gin.Context) {
	namespace := g.Param("namespace")
	name := g.Param("name")
	item, err := w.daemonSet.Get(namespace, name)
	if err != nil {
		g.JSON(http.StatusBadRequest,
			gin.H{code: http.StatusBadRequest, data: "", msg: err.Error(), status: "Request bad parameter"})
		return
	}
	g.JSON(http.StatusOK, item)
}

// List DaemonSet
func (w *WorkloadsAPI) ListDaemonSet(g *gin.Context) {
	list, _ := w.daemonSet.List("", "", 0, 10000, nil)
	daemonSetList := &v1.DaemonSetList{}
	marshalData, err := json.Marshal(list)
	if err != nil {
		g.JSON(http.StatusBadRequest,
			gin.H{code: http.StatusBadRequest, data: "", msg: err.Error(), status: "Request bad parameter"})
		return
	}
	_ = json.Unmarshal(marshalData, daemonSetList)
	g.JSON(http.StatusOK, daemonSetList)
}
