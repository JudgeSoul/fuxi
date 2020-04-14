package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/yametech/fuxi/pkg/api/workload/template"
	dyn "github.com/yametech/fuxi/pkg/kubernetes/client"
	appsv1 "k8s.io/api/apps/v1"

	"net/http"
)

func (w *WorkloadsAPI) GetDeployment(g *gin.Context) {
	namespace := g.Param("namespace")
	name := g.Param("name")
	item, err := w.deployments.Get(dyn.ResourceDeployment, namespace, name)
	if err != nil {
		panic(err)
	}
	g.JSON(http.StatusOK, item)
}

func (w *WorkloadsAPI) ListDeployment(g *gin.Context) {
	list, _ := w.deployments.List(dyn.ResourceDeployment, "", "", 0, 100, nil)
	deploymentList := &appsv1.DeploymentList{}
	data, err := json.Marshal(list)
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(data, deploymentList)
	g.JSON(http.StatusOK, deploymentList)
}

func (w *WorkloadsAPI) ApplyDeployment(g *gin.Context) {
	deploymentRequest := &template.DeploymentRequest{}
	if err := g.ShouldBind(deploymentRequest); err != nil {
		g.JSON(http.StatusBadRequest,
			gin.H{
				code:   http.StatusBadRequest,
				data:   "",
				msg:    err.Error(),
				status: "Request bad parameter",
			},
		)
		return
	}
}

func (w *WorkloadsAPI) DeleteDeployment(g *gin.Context) {}