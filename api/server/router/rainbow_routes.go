package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/caoyingjunz/pixiulib/httputils"
	"github.com/caoyingjunz/rainbow/pkg/types"
)

func (cr *rainbowRouter) createLabel(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateLabelRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateLabel(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteLabel(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteLabel(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateLabel(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req    types.UpdateLabelRequest
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateLabel(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listLabels(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListLabels(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listLabelImages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListLabelImages(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listRainbowds(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err        error
		listOption types.ListOptions
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListRainbowds(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createTask(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateTaskRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateTask(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createTaskV2(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateTaskRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateTaskV2(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateTask(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req    types.UpdateTaskRequest
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateTask(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) UpdateTaskStatus(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		req    types.UpdateTaskStatusRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.TaskId = idMeta.ID
	if err = cr.c.Server().UpdateTaskStatus(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteTask(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if err = cr.c.Server().DeleteTask(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) reRunTask(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.UpdateTaskRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().ReRunTask(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getTask(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if resp.Result, err = cr.c.Server().GetTask(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listTasks(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListTasks(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listTasksByIds(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		ids struct {
			Ids []int64 `json:"ids"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &ids, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListTasksByIds(c, ids.Ids); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteTasksByIds(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		ids struct {
			Ids []int64 `json:"ids"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &ids, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteTasksByIds(c, ids.Ids); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listTaskImages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta     types.IdMeta
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListTaskImages(c, idMeta.ID, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createRegistry(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateRegistryRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateRegistry(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) loginRegistry(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateRegistryRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().LoginRegistry(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateRegistry(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		req    types.UpdateRegistryRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateRegistry(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteRegistry(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if err = cr.c.Server().DeleteRegistry(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getRegistry(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if resp.Result, err = cr.c.Server().GetRegistry(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listRegistries(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err        error
		listOption types.ListOptions
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if resp.Result, err = cr.c.Server().ListRegistries(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createAgent(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateAgentRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateAgent(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteAgent(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteAgent(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateAgent(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			Name string `uri:"Name" binding:"required"`
		}
		req types.UpdateAgentRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.AgentName = idMeta.Name
	if err = cr.c.Server().UpdateAgent(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getAgent(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetAgent(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateAgentStatus(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.UpdateAgentStatusRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().UpdateAgentStatus(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createAgentRepo(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		githubReq types.CallGithubRequest
		idMeta    struct {
			Name string `uri:"Name" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &githubReq, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	githubReq.ClientId = idMeta.Name
	if resp.Result, err = cr.c.Server().CreateAgentRepo(c, &githubReq); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listAgents(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListAgents(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createImage(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateImageRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if err = cr.c.Server().CreateImage(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createImages(c *gin.Context) {
	resp := types.Response{}

	var (
		req types.CreateImagesRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	if resp.Result, err = cr.c.Server().CreateImages(c, &req); err != nil {
		resp.Code = http.StatusBadRequest
		resp.Message = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Code = 200
	c.JSON(http.StatusOK, resp)
}

func (cr *rainbowRouter) listImagesByIds(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		ids struct {
			Ids []int64 `json:"ids"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &ids, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListImagesByIds(c, ids.Ids); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteImagesByIds(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		ids struct {
			Ids []int64 `json:"ids"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &ids, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteImagesByIds(c, ids.Ids); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateImage(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		req    types.UpdateImageRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateImage(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteImage(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteImage(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteImageTag(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID    int64 `uri:"Id" binding:"required"`
			TagId int64 `uri:"TagId" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteImageTag(c, idMeta.ID, idMeta.TagId); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) bindImageLabels(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID int64 `uri:"Id" binding:"required"`
		}
		req types.BindImageLabels
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().BindImageLabels(c, idMeta.ID, req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getImageTag(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID    int64 `uri:"Id" binding:"required"`
			TagId int64 `uri:"TagId" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetImageTag(c, idMeta.ID, idMeta.TagId); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listImageLabels(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID int64 `uri:"Id" binding:"required"`
		}
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListImageLabels(c, idMeta.ID, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listImageTags(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID int64 `uri:"Id" binding:"required"`
		}
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListImageTags(c, idMeta.ID, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) UpdateImageStatus(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.UpdateImageStatusRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	if err = cr.c.Server().UpdateImageStatus(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getImage(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetImage(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) searchRepo(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().SearchRepo(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listImages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListImages(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listImagesForClient(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListImagesForClient(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getCollections(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetCollection(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) AddDailyReview(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		pageOption struct {
			Page string `json:"page"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &pageOption, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().AddDailyReview(c, pageOption.Page); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getDailyMetrics(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListMetrics(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getUserResources(c *gin.Context) {
	resp := httputils.NewResponse()

	claims := GetLoginUser(c)
	if claims == nil {
		httputils.SetFailed(c, resp, fmt.Errorf("未登录"))
		return
	}
	var err error
	if resp.Result, err = cr.c.Server().GetUserResources(c, claims.UserId, claims.Role); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getUserInfoByAccessKey(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetUserInfoByAccessKey(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createLogo(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateLogoRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateLogo(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteLogo(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteLogo(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateLogo(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		idMeta types.IdMeta
		req    types.UpdateLogoRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateLogo(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listLogos(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListLogos(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) overview(c *gin.Context) {
	resp := httputils.NewResponse()

	var err error
	if resp.Result, err = cr.c.Server().Overview(c); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) store(c *gin.Context) {
	resp := httputils.NewResponse()
	var err error
	if resp.Result, err = cr.c.Server().Store(c); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) downflow(c *gin.Context) {
	resp := httputils.NewResponse()
	var err error
	if resp.Result, err = cr.c.Server().Downflow(c); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getImageDownflow(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		downflowMeta types.DownflowMeta
		err          error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &downflowMeta); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ImageDownflow(c, downflowMeta); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) searchImages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().SearchImages(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createNamespace(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateNamespaceRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateNamespace(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateNamespace(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		idMeta types.IdMeta
		req    types.UpdateNamespaceRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateNamespace(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteNamespace(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteNamespace(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listNamespaces(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListNamespaces(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) searchRepositories(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CallSearchRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if resp.Result, err = cr.c.Server().SearchRepositories(c, req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) searchRepositoryTags(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CallSearchRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if resp.Result, err = cr.c.Server().SearchRepositoryTags(c, req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getRepositoryTagInfo(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req      types.CallSearchRequest
		nameMeta types.NameMeta
		err      error
	)
	if err = httputils.ShouldBindAny(c, nil, &nameMeta, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Repository = nameMeta.Name
	req.Namespace = nameMeta.Namespace
	req.Tag = c.Param("tag")
	if resp.Result, err = cr.c.Server().GetRepositoryTagInfo(c, req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createSubscribe(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		req types.CreateSubscribeRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateSubscribe(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listSubscribes(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListSubscribes(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listSubscribeMessages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListSubscribeMessages(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) runSubscribeNow(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.RunSubscribeRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().RunSubscribe(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateSubscribe(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		req    types.UpdateSubscribeRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateSubscribe(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getSubscribe(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetSubscribe(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteSubscribe(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteSubscribe(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createTaskMessage(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		req    types.CreateTaskMessageRequest
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().CreateTaskMessage(c, req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listTaskMessages(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListTaskMessages(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listArchitectures(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err error
	)
	if resp.Result, err = cr.c.Server().ListArchitectures(c, types.ListOptions{}); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createUser(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateUserRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateUser(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createOrUpdateUsers(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateUsersRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateOrUpdateUsers(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateUser(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID string `uri:"Id" binding:"required"`
		}
		req types.UpdateUserRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	req.UserId = idMeta.ID
	if err = cr.c.Server().UpdateUser(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteUser(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID string `uri:"Id" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteUser(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)

}

func (cr *rainbowRouter) getUser(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID string `uri:"Id" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetUser(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listUsers(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListUsers(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createAccess(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateAccessRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	// 返回创建结果（含一次性明文 SK）
	if resp.Result, err = cr.c.Server().CreateAccess(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteAccess(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			Ak string `uri:"accessKey" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteAccess(c, idMeta.Ak); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listAccesses(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListAccesses(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createNotification(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateNotificationRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().CreateNotify(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateNotification(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.UpdateNotificationRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().UpdateNotify(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteNotification(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteNotify(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getNotification(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().GetNotify(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listNotifications(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListNotifies(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) enableNotify(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.UpdateNotificationRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().EnableNotify(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) getNotificationTypes(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		err error
	)
	if resp.Result, err = cr.c.Server().GetNotifyTypes(c); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) sendNotification(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.SendNotificationRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().SendNotify(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listKubernetesTags(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err        error
		listOption types.ListOptions
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListKubernetesVersions(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) syncKubernetesTags(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err error
		req types.CallKubernetesTagRequest
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().SyncKubernetesTags(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) syncAgentDrivers(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err error
		req types.CallGithubRequest
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().SyncAgentRepos(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) syncNamespaces(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		err error
		req types.SyncNamespaceRequest
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().SyncNamespace(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) fix(c *gin.Context) {
	resp := httputils.NewResponse()
	var (
		err error
		req types.FixRequest
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().Fix(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listRoles(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListRoles(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) createRole(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		req types.CreateRoleRequest
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, nil, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().CreateRole(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) updateRole(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		req    types.UpdateRoleRequest
		err    error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	req.Id = idMeta.ID
	if err = cr.c.Server().UpdateRole(c, &req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) deleteRole(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta types.IdMeta
		err    error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().DeleteRole(c, idMeta.ID); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listPermissions(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListPermissions(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) listResources(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		listOption types.ListOptions
		err        error
	)
	if err = httputils.ShouldBindAny(c, nil, nil, &listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if resp.Result, err = cr.c.Server().ListResources(c, listOption); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) assignUserRole(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID string `uri:"Id" binding:"required"`
		}
		req struct {
			RoleId int64 `json:"role_id" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, &req, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().AssignUserRole(c, idMeta.ID, req.RoleId); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}

func (cr *rainbowRouter) removeUserRole(c *gin.Context) {
	resp := httputils.NewResponse()

	var (
		idMeta struct {
			ID     string `uri:"Id" binding:"required"`
			RoleId int64  `uri:"RoleId" binding:"required"`
		}
		err error
	)
	if err = httputils.ShouldBindAny(c, nil, &idMeta, nil); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}
	if err = cr.c.Server().RemoveUserRole(c, idMeta.ID, idMeta.RoleId); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	httputils.SetSuccess(c, resp)
}
