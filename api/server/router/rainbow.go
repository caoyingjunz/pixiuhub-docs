package router

import (
	"net/http"

	"github.com/caoyingjunz/pixiulib/httputils"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/cmd/app/options"
	"github.com/caoyingjunz/rainbow/pkg/controller"
	"github.com/caoyingjunz/rainbow/pkg/types"
)

type rainbowRouter struct {
	c controller.RainbowInterface
}

// NewRouter initializes a new Task router
func NewRouter(o *options.ServerOptions) {
	s := &rainbowRouter{
		c: o.Controller,
	}
	s.initRoutes(o.HttpEngine)
}

// login 账号密码登录，签发 JWT
func (cr *rainbowRouter) login(c *gin.Context) {
	resp := httputils.NewResponse()

	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputils.SetFailed(c, resp, err)
		return
	}

	loginResp, err := cr.c.Server().Login(c, &req)
	if err != nil {
		klog.Errorf("用户(%s)登录失败: %v", req.Name, err)
		httputils.SetFailed(c, resp, err)
		return
	}

	resp.Result = loginResp
	httputils.SetSuccess(c, resp)
	c.Status(http.StatusOK)
}

func (cr *rainbowRouter) initRoutes(httpEngine *gin.Engine) {
	// v2
	routeV2 := httpEngine.Group("/api/v2")
	{
		// 指标
		metricsRoute := routeV2.Group("/metrics")
		{
			metricsRoute.GET("/active-users/daily", cr.getDailyMetrics)
		}

		// 通过 ak 获取用户信息
		userRouteV2 := routeV2.Group("/users")
		{
			userRouteV2.GET("", cr.getUserInfoByAccessKey)
			// 账号密码登录，签发 JWT
			userRouteV2.POST("/login", cr.login)
			// 当前用户可见资源（菜单/按钮权限）
			userRouteV2.GET("/resources", cr.getUserResources)
		}

		registryRouteV2 := routeV2.Group("/registries")
		{
			registryRouteV2.GET("", cr.listRegistries)
		}

		// 角色管理（RBAC）
		roleRouteV2 := routeV2.Group("/roles")
		{
			roleRouteV2.GET("", cr.listRoles)
			roleRouteV2.POST("", cr.createRole)
			roleRouteV2.PUT("/:Id", cr.updateRole)
			roleRouteV2.DELETE("/:Id", cr.deleteRole)
		}

		// 权限与资源管理（RBAC）
		permissionRouteV2 := routeV2.Group("/permissions")
		{
			permissionRouteV2.GET("", cr.listPermissions)
		}
		resourceRouteV2 := routeV2.Group("/resources")
		{
			resourceRouteV2.GET("", cr.listResources)
		}

		// 任务
		taskV2Route := routeV2.Group("/tasks")
		{
			taskV2Route.POST("", cr.createTaskV2)
		}

		// 镜像
		imageRoute := routeV2.Group("/images")
		{
			imageRoute.GET("/:Id", cr.getImage)
			imageRoute.GET("", cr.listImagesForClient)
		}

		searchRoute := routeV2.Group("/search")
		{
			// 直接 pull 搜索 tag
			searchRoute.GET("/repos", cr.searchRepo)

			// 远端搜索，支持 dockerhub
			searchRoute.GET("/repositories", cr.searchRepositories)
			searchRoute.GET("/repositories/tags", cr.searchRepositoryTags)
			searchRoute.GET("/repositories/:namespace/:name/tags/:tag", cr.getRepositoryTagInfo)
		}

		// 客户端下载
		pixiuctlRoute := routeV2.Group("/pixiuctls")
		{
			pixiuctlRoute.GET("", cr.listPixiuctls)
			pixiuctlRoute.GET("/:version/:filename", cr.downloadPixiuctl)
		}
	}

	// v1
	taskRoute := httpEngine.Group("/rainbow/tasks")
	{
		taskRoute.POST("", cr.createTask)
		taskRoute.PUT("/:Id", cr.updateTask)
		taskRoute.DELETE("/:Id", cr.deleteTask)
		taskRoute.GET("/:Id", cr.getTask)
		taskRoute.GET("", cr.listTasks)

		taskRoute.PUT("/:Id/status", cr.UpdateTaskStatus) // DEPRECATED
		taskRoute.GET(":Id/images", cr.listTaskImages)
		taskRoute.POST("/rerun", cr.reRunTask)

		taskRoute.POST("/:Id/messages", cr.createTaskMessage)
		taskRoute.GET(":Id/messages", cr.listTaskMessages)
	}

	archRoute := httpEngine.Group("/rainbow/architectures")
	{
		archRoute.GET("", cr.listArchitectures)
	}

	subscribeRoute := httpEngine.Group("/rainbow/subscribes")
	{
		subscribeRoute.POST("", cr.createSubscribe)
		subscribeRoute.PUT("/:Id", cr.updateSubscribe)
		subscribeRoute.DELETE("/:Id", cr.deleteSubscribe)
		subscribeRoute.GET("/:Id", cr.getSubscribe)
		subscribeRoute.GET("", cr.listSubscribes)

		subscribeRoute.GET("/:Id/messages", cr.listSubscribeMessages)
	}

	// 执行 API 路由
	runRoute := httpEngine.Group("/rainbow/run")
	{
		runRoute.POST("/subscribes", cr.runSubscribeNow)
	}

	kubernetesVersionRoute := httpEngine.Group("/rainbow/kubernetes/tags")
	{
		kubernetesVersionRoute.GET("", cr.listKubernetesTags)
	}

	registryRoute := httpEngine.Group("/rainbow/registries")
	{
		registryRoute.POST("", cr.createRegistry)
		registryRoute.POST("/login", cr.loginRegistry)
		registryRoute.PUT("/:Id", cr.updateRegistry)
		registryRoute.DELETE("/:Id", cr.deleteRegistry)
		registryRoute.GET("/:Id", cr.getRegistry)
		registryRoute.GET("", cr.listRegistries)
	}

	agentRoute := httpEngine.Group("/rainbow/agents")
	{
		agentRoute.POST("", cr.createAgent)
		agentRoute.PUT("/:Name", cr.updateAgent)
		agentRoute.DELETE("/:Id", cr.deleteAgent)
		agentRoute.GET("/:Id", cr.getAgent)
		agentRoute.GET("", cr.listAgents)
		agentRoute.PUT("/status", cr.updateAgentStatus)

		// 指定 agent 创建 repo
		agentRoute.POST("/:Name/repos", cr.createAgentRepo)
	}

	imageRoute := httpEngine.Group("/rainbow/images")
	{
		imageRoute.POST("", cr.createImage)
		imageRoute.PUT("/:Id", cr.updateImage)
		imageRoute.DELETE("/:Id", cr.deleteImage)
		imageRoute.GET("/:Id", cr.getImage)
		imageRoute.GET("", cr.listImages)

		imageRoute.PUT("/status", cr.UpdateImageStatus)
		imageRoute.POST("/batches", cr.createImages)

		imageRoute.GET("/:Id/tags", cr.listImageTags)
		imageRoute.DELETE("/:Id/tags/:TagId", cr.deleteImageTag)
		imageRoute.GET("/:Id/tags/:TagId", cr.getImageTag)

		// 镜像关联 Label API
		imageRoute.POST("/:Id/labels", cr.bindImageLabels)
		imageRoute.GET("/:Id/labels", cr.listImageLabels)
	}

	batchRoute := httpEngine.Group("/rainbow/batch")
	{
		batchRoute.POST("/list/images", cr.listImagesByIds)     // 根据ids批量获取镜像列表
		batchRoute.POST("/delete/images", cr.deleteImagesByIds) // 根据ids批量删除镜像列表

		batchRoute.POST("/list/tasks", cr.listTasksByIds)
		batchRoute.POST("/delete/tasks", cr.deleteTasksByIds)
	}

	searchRoute := httpEngine.Group("/rainbow/search")
	{
		searchRoute.GET("/images", cr.searchImages)
	}

	//collectRoute := httpEngine.Group("/rainbow/collections")
	//{
	//	collectRoute.GET("", cr.getCollections)
	//	collectRoute.POST("/add/review", cr.AddDailyReview)
	//}

	labelRoute := httpEngine.Group("/rainbow/labels")
	{
		labelRoute.POST("", cr.createLabel)
		labelRoute.DELETE("/:Id", cr.deleteLabel)
		labelRoute.PUT("/:Id", cr.updateLabel)
		labelRoute.GET("", cr.listLabels)

		labelRoute.GET("/images", cr.listLabelImages) // 根据标签获取镜像列表，标签可以是多个，格式 ?labels=labelId1,LABLiD2
	}

	logoRoute := httpEngine.Group("/rainbow/logos")
	{
		logoRoute.POST("", cr.createLogo)
		logoRoute.DELETE("/:Id", cr.deleteLogo)
		logoRoute.PUT("/:Id", cr.updateLogo)
		logoRoute.GET("", cr.listLogos)
	}

	nsRoute := httpEngine.Group("/rainbow/namespaces")
	{
		nsRoute.POST("", cr.createNamespace)
		nsRoute.PUT("/:Id", cr.updateNamespace)
		nsRoute.DELETE("/:Id", cr.deleteNamespace)
		nsRoute.GET("", cr.listNamespaces)
	}

	userRoute := httpEngine.Group("/rainbow/users")
	{
		userRoute.POST("", cr.createUser)
		userRoute.PUT("/:Id", cr.updateUser)
		userRoute.DELETE("/:Id", cr.deleteUser)
		userRoute.GET("/:Id", cr.getUser)
		userRoute.GET("", cr.listUsers)
		// 用户-角色绑定
		userRoute.POST("/:Id/roles", cr.assignUserRole)
		userRoute.DELETE("/:Id/roles/:RoleId", cr.removeUserRole)
	}

	accessRoute := httpEngine.Group("/rainbow/access")
	{
		accessRoute.POST("", cr.createAccess)
		accessRoute.DELETE("/:accessKey", cr.deleteAccess)
		accessRoute.GET("", cr.listAccesses)
	}

	syncRoute := httpEngine.Group("/rainbow/sync")
	{
		syncRoute.POST("/users", cr.createOrUpdateUsers)
		syncRoute.POST("/kubernetes/tags", cr.syncKubernetesTags)
		syncRoute.POST("/agents/drivers", cr.syncAgentDrivers)
		syncRoute.POST("/namespaces", cr.syncNamespaces)
	}

	// 镜像汇总
	overviewRoute := httpEngine.Group("/rainbow/overview")
	{
		overviewRoute.GET("", cr.overview)
		overviewRoute.GET("/downflow/daily", cr.downflow)
		overviewRoute.GET("/store/daily", cr.store)
		overviewRoute.GET("/image/daily", cr.getImageDownflow)
	}

	repoRoute := httpEngine.Group("/rainbow/search")
	{
		repoRoute.GET("/repositories", cr.searchRepositories)
		repoRoute.GET("/repositories/tags", cr.searchRepositoryTags)
		repoRoute.GET("/repositories/:namespace/:name/tags/:tag", cr.getRepositoryTagInfo)
	}

	notifyRoute := httpEngine.Group("/rainbow/notifications")
	{
		notifyRoute.POST("", cr.createNotification)
		notifyRoute.PUT("/:Id", cr.updateNotification)
		notifyRoute.DELETE("/:Id", cr.deleteNotification)
		notifyRoute.GET("/:Id", cr.getNotification)
		notifyRoute.GET("", cr.listNotifications)

		notifyRoute.POST("/enable/notify", cr.enableNotify)
	}

	notifyTypeRoute := httpEngine.Group("/rainbow/notification/types")
	{
		notifyTypeRoute.GET("", cr.getNotificationTypes)
	}

	sendNotifyRoute := httpEngine.Group("/rainbow/send/notification")
	{
		sendNotifyRoute.POST("", cr.sendNotification)
	}

	fixRoute := httpEngine.Group("/rainbow/fix")
	{
		fixRoute.POST("", cr.fix)
	}

	chartRoute := httpEngine.Group("/rainbow/chartrepo")
	{
		chartRoute.POST("/enable", cr.enableChartRepo) // 启用 helm chart repo
		chartRoute.GET("/:project/status", cr.getChartRepoStatus)

		chartRoute.GET("/:project/charts", cr.ListCharts)
		chartRoute.GET("/:project/charts/:chart", cr.ListChartVersions)
		chartRoute.DELETE("/:project/charts/:chart", cr.DeleteChart)

		chartRoute.GET("/:project/token", cr.getToken)

		// 上传 chart 到指定项目
		chartRoute.POST("/upload/:project", cr.uploadChart)
		// 下载 chart
		chartRoute.GET("/download/:project/charts/:chart/:version", cr.downloadChart)

		chartRoute.GET("/:project/charts/:chart/:version", cr.GetChartVersion)
		chartRoute.DELETE("/:project/charts/:chart/:version", cr.DeleteChartVersion)
	}

	// 构建镜像 API
	buildRoute := httpEngine.Group("/rainbow/builds")
	{
		buildRoute.POST("", cr.createBuild)
		buildRoute.DELETE("/:Id", cr.deleteBuild)
		buildRoute.PUT("/:Id", cr.updateBuild)
		buildRoute.GET("", cr.listBuilds)
		buildRoute.GET("/:Id", cr.getBuild)

		buildRoute.POST("/:Id/messages", cr.createBuildMessage)
		buildRoute.GET("/:Id/messages", cr.listBuildMessages)
	}

	rainbowdRoute := httpEngine.Group("/rainbow/rainbowds")
	{
		rainbowdRoute.GET("", cr.listRainbowds)
	}

	// 设置资源状态API
	setStatus := httpEngine.Group("/rainbow/set")
	{
		setStatus.PUT("/build/:Id/status", cr.setBuildStatus)
		setStatus.PUT("/task/:Id/status", cr.UpdateTaskStatus)
	}
}
