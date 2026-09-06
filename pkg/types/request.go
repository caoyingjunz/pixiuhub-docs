package types

import "time"

type (
	UserMetaRequest struct {
		UserId   string `json:"user_id"`
		UserName string `json:"user_name"`
	}

	IdMetaRequest struct {
		Id              int64 `json:"id"`
		ResourceVersion int64 `json:"resource_version"`
	}

	ChartMetaRequest struct {
		Project string `uri:"project" binding:"required" form:"project"`
		Chart   string `uri:"chart" form:"chart"`
		Version string `uri:"version" form:"version"`
	}

	CreateBuildRequest struct {
		UserId         string `json:"user_id"`
		Name           string `json:"name"`
		Arch           string `json:"arch"`        // 架构
		Dockerfile     string `json:"dockerfile"`  // 镜像构建 dockerfile
		RegistryId     int64  `json:"registry_id"` // 推送镜像仓库
		AgentName      string `json:"agent_name"`  // 执行代理
		Namespace      string `json:"namespace"`
		Repo           string `json:"repo"` // 构建完成后的镜像名称
		DockerfilePath string `yaml:"dockerfile_path"`
	}

	UpdateBuildRequest struct {
		Id              int64  `json:"id"`
		Status          string `json:"status"`
		ResourceVersion int64  `json:"resource_version"`

		CreateBuildRequest `json:",inline"`
	}

	CreateLabelRequest struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	CreateLogoRequest struct {
		Name string `json:"name" binding:"required"`
		Logo string `json:"logo"`
	}

	UpdateLogoRequest struct {
		Id              int64  `json:"id"`
		ResourceVersion int64  `json:"resource_version"`
		Name            string `json:"name" binding:"required"`
		Logo            string `json:"logo"`
	}

	UpdateLabelRequest struct {
		Id              int64  `json:"id"`
		ResourceVersion int64  `json:"resource_version"`
		Name            string `json:"name" binding:"required"`
		Description     string `json:"description"`
	}

	CreateTaskRequest struct {
		Name              string   `json:"name"`
		UserId            string   `json:"user_id"`
		UserName          string   `json:"user_name"`
		RegisterId        int64    `json:"register_id"`
		Type              int      `json:"type"` // 0：直接指定镜像列表 1: 指定 kubernetes 版本
		KubernetesVersion string   `json:"kubernetes_version"`
		Images            []string `json:"images"` // 镜像列表中含镜像和架构，格式类似 nginx:1.0.1/amd64，直接指定架构优先级高于任务 arch
		AgentName         string   `json:"agent_name"`
		Mode              int64    `json:"mode"`
		PublicImage       bool     `json:"public_image"`
		Driver            string   `json:"driver"`
		Logo              string   `json:"logo"`
		Namespace         string   `json:"namespace"`
		IsOfficial        bool     `json:"is_official"`
		Architecture      string   `json:"architecture"`
		OwnerRef          int      `json:"owner_ref"` // 任务所属，直接创建 0，订阅创建 1
		SubscribeId       int64    `json:"subscribe_id"`
	}

	UpdateTaskRequest struct {
		Id                int64    `json:"id"`
		ResourceVersion   int64    `json:"resource_version"`
		Name              string   `json:"name"`
		UserId            string   `json:"user_id"`
		RegisterId        int64    `json:"register_id"`
		Type              int      `json:"type"` // 0：直接指定镜像列表 1: 指定 kubernetes 版本
		KubernetesVersion string   `json:"kubernetes_version"`
		AgentName         string   `json:"agent_name"`
		Status            string   `json:"status"`
		Images            []string `json:"images"`
		Mode              int64    `json:"mode"`
		PublicImage       bool     `json:"public_image"`
		OnlyPushError     bool     `json:"only_push_error"` // 仅同步推送异常
	}

	UpdateTaskStatusRequest struct {
		TaskId  int64  `json:"task_id"`
		Status  string `json:"status"`
		Message string `json:"message"`
		Process int    `json:"process"`
	}

	UpdateBuildStatusRequest struct {
		BuildId int64  `json:"build_id"`
		Status  string `json:"status"`
	}

	CreateRegistryRequest struct {
		Name       string `json:"name"`
		UserId     string `json:"user_id"`
		Repository string `json:"repository"`
		Namespace  string `json:"namespace"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		Role       int    `json:"role"`
	}

	UpdateRegistryRequest struct {
		Id              int64  `json:"id"`
		ResourceVersion int64  `json:"resource_version"`
		Name            string `json:"name"`
		UserId          string `json:"user_id"`
		Repository      string `json:"repository"`
		Namespace       string `json:"namespace"`
		Username        string `json:"username"`
		Password        string `json:"password"`
	}

	CreateImageRequest struct {
		TaskId     int64  `json:"task_id"`
		TaskName   string `json:"task_name"`
		UserId     string `json:"user_id"`
		RegisterId int64  `json:"register_id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		IsPublic   bool   `json:"is_public"`
		IsLocked   bool   `json:"is_locked"`
		Arch       string `json:"arch"`
	}

	CreateImagesRequest struct {
		TaskId   int64    `json:"task_id"`
		TaskName string   `json:"task_name"`
		Names    []string `json:"names"`
	}

	UpdateImageRequest struct {
		Id              int64  `json:"id"`
		ResourceVersion int64  `json:"resource_version"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		Label           string `json:"label"`
		IsPublic        bool   `json:"is_public"`
		Logo            string `json:"logo"`
		Description     string `json:"description"`
		IsLocked        bool   `json:"is_locked"`
	}

	UpdateImageStatusRequest struct {
		Name       string `json:"name"` // 对应仓里的镜像名称 比如，nginx
		ImageId    int64  `json:"image_id"`
		TaskId     int64  `json:"task_id"`
		RegistryId int64  `json:"registry_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		Target     string `json:"target"`
	}

	CreateAccessRequest struct {
		UserId     string  `json:"user_id" binding:"required"`
		UserName   string  `json:"user_name"`
		ExpireTime *string `json:"expire_time"`
	}

	BindImageLabels struct {
		OP       int     `json:"op"` // add or delete, 默认添加
		LabelIds []int64 `json:"label_ids"`
	}

	CreateNamespaceRequest struct {
		Name        string `json:"name"`
		Label       string `json:"label"`
		Logo        string `json:"logo"`
		Description string `json:"description"`
	}

	UpdateNamespaceRequest struct {
		Id              int64 `json:"id"`
		ResourceVersion int64 `json:"resource_version"`

		Name        string `json:"name"`
		Label       string `json:"label"`
		Logo        string `json:"logo"`
		Description string `json:"description"`
	}

	SyncNamespaceRequest struct {
		Id int64 `json:"id"`

		SyncType int  `json:"sync_type"` // 0 logo 1 label 2 默认logo
		Rewrite  bool `json:"rewrite"`   // 覆盖现有配置
	}

	CreateUserRequest struct {
		Name        string  `json:"name"`
		UserId      string  `json:"user_id"`
		Password    string  `json:"password"`  // 创建用户时设置初始密码（bcrypt 存储）
		UserType    int     `json:"user_type"` // 个人版，专有版
		PaymentType int     `json:"payment_type"`
		ExpireTime  *string `json:"expire_time"` // payment_type为 0 时无需设置
		RemainCount int     `json:"remain_count"`
		Sync        bool    `json:"sync"`
		Role        int     `json:"role"`
		Email       string  `json:"email"`
	}

	UpdateUserRequest struct {
		ResourceVersion int64 `json:"resource_version"`

		CreateUserRequest `json:",inline"`
	}

	CreateUsersRequest struct {
		Users []CreateUserRequest
	}

	CreateAgentRequest struct {
		AgentName        string `json:"agent_name"`
		Type             string `json:"type"`
		GithubUser       string `json:"github_user"`       // github 后端用户名
		GithubRepository string `json:"github_repository"` // github 仓库地址
		GithubToken      string `json:"github_token"`      // github token
		GithubEmail      string `json:"github_email"`
		RainbowdName     string `json:"rainbowd_name"`
	}

	UpdateAgentRequest struct {
		AgentName string `json:"agent_name"`

		GithubUser       string `json:"github_user"`       // github 后端用户名
		GithubRepository string `json:"github_repository"` // github 仓库地址
		GithubToken      string `json:"github_token"`      // github token
		GithubEmail      string `json:"github_email"`
		RainbowdName     string `json:"rainbowd_name"`
	}

	UpdateAgentStatusRequest struct {
		AgentName string `json:"agent_name"`
		Status    string `json:"status"`
	}

	CreateNotificationRequest struct {
		UserMetaRequest `json:",inline"`

		Name      string      `json:"name"`
		Role      int         `json:"role"` // 1 管理员 0 普通用户
		Enable    bool        `json:"enable"`
		Type      string      `json:"type"` // 支持 webhook, dingtalk, wecom
		PushCfg   *PushConfig `json:"push_cfg,omitempty"`
		ShortDesc string      `json:"short_desc"`
	}

	UpdateNotificationRequest struct {
		IdMetaRequest   `json:",inline"`
		UserMetaRequest `json:",inline"`

		Name      string      `json:"name"`
		Role      int         `json:"role"` // 1 管理员 0 普通用户
		Enable    bool        `json:"enable"`
		Type      string      `json:"type"` // 支持 webhook, dingtalk, wecom
		PushCfg   *PushConfig `json:"push_cfg,omitempty"`
		ShortDesc string      `json:"short_desc"`
	}

	NotificationResult struct {
		Id              int64     `json:"id"`
		GmtCreate       time.Time `json:"gmt_create"`
		GmtModified     time.Time `json:"gmt_modified"`
		ResourceVersion int64     `json:"resource_version"`

		CreateNotificationRequest `json:",inline"`
	}

	SendNotificationRequest struct {
		CreateNotificationRequest `json:",inline"`

		Email   string `json:"email"`   // 系统通知，用户注册时使用
		DryRun  bool   `json:"dry_run"` // 测试连通性
		Content string `json:"content"` // 用户通知，用于存储镜像内容
	}

	FixRequest struct {
		Type   string       `json:"type"` // 修复资源类型
		UserId string       `json:"user_id"`
		Image  FixImageSpec `json:"image"`
		Tag    FixTagSpec   `json:"tag"`
	}

	FixImageSpec struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	FixTagSpec struct {
		Method string `json:"method"`
	}

	// PageRequest 分页配置
	PageRequest struct {
		Page  int `form:"page" json:"page"`   // 页数，表示第几页
		Limit int `form:"limit" json:"limit"` // 每页数量，表示每页几个对象
	}
	// QueryOption 搜索配置
	QueryOption struct {
		LabelSelector string `form:"labelSelector" json:"labelSelector"` // 标签搜索
		NameSelector  string `form:"nameSelector" json:"nameSelector"`   // 名称搜索
	}

	CustomMeta struct {
		Status       int    `form:"status"`
		Namespace    string `form:"namespace"`
		Agent        string `form:"agent"`
		OwnerRef     string `form:"ownerRef"`
		SubscribeId  int64  `form:"subscribe_id"`
		Project      string `form:"project"`
		AgentStatus  string `form:"agent_status"`
		CustomStatus string `form:"custom_status"`
		LabelIds     string `form:"label_ids"`
		Trusted      int    `form:"trusted"`
		ExtendLimit  int    `form:"extend_limit"`
		Arch         string `form:"arch"`
		AccessKey    string `form:"access_key"`
		Role         int    `form:"role"`
		SortBy       string `form:"sort_by"`
		UserType     int    `form:"user_type"`
		PaymentType  int    `form:"payment_type"` // 付费模式 0 按量付费， 1 包年包月
	}

	RemoteSearchRequest struct {
		Hub       string `json:"hub" form:"hub"`
		ClientId  string `json:"client_id" form:"client_id"` // 指定后端执行 clientId
		Namespace string `json:"namespace" form:"namespace"`
		Query     string `json:"query" form:"query"`
		Page      string `json:"page" form:"page"`
		PageSize  string `json:"page_size" form:"page_size"`
	}

	RemoteTagSearchRequest struct {
		Hub        string        `json:"hub" form:"hub"`
		ClientId   string        `json:"client_id" form:"client_id"`
		Namespace  string        `json:"namespace" form:"namespace"`
		Repository string        `json:"repository" form:"repository"` // repo_name <namespace>/<repository_name>
		Query      string        `json:"query" form:"query"`
		Page       int           `json:"page" form:"page"`
		PageSize   int           `json:"page_size" form:"page_size"`
		SearchType int           `json:"search_type" form:"search_type"` // 0 模糊搜索 1 精准查询
		Config     *SearchConfig `json:"config"`
	}

	SearchConfig struct {
		Page      int    `json:"page"`
		Size      int    `json:"size"`
		ImageFrom string `json:"image_from"`
		Policy    string `json:"policy"`
		Arch      string `json:"arch"`
	}

	RemoteTagInfoSearchRequest struct {
		Hub        string `json:"hub" form:"hub"`
		ClientId   string `json:"client_id" form:"client_id"`
		Namespace  string `json:"namespace" form:"namespace"`
		Repository string `json:"repository" form:"repository"`
		Tag        string `json:"tag" form:"tag"`
	}

	CallKubernetesTagRequest struct {
		ClientId string `json:"client_id"`
		SyncAll  bool   `json:"sync_all"`
	}

	CallGithubRequest struct {
		ClientId string `json:"client_id,omitempty"`

		Op    int      `json:"op,omitempty"` // 操作类型
		Repo  string   `json:"repo,omitempty"`
		Repos []string `json:"repos,omitempty"`
	}

	CallSearchRequest struct {
		ClientId string `json:"client_id" form:"client_id"` // 指定后端执行 clientId

		Hub        string `json:"hub" form:"hub"`
		SearchType int    `json:"search_type" form:"search_type"` // 搜索类型，镜像搜索，kubernetesTag 等
		TargetType int    `json:"target_type" form:"target_type"` // 目标类型，搜索镜像，搜索镜像版本，查询镜像详情

		Namespace  string `json:"namespace" form:"namespace"`
		Repository string `json:"repository" form:"repository"`
		Tag        string `json:"tag" form:"tag"`

		Query    string `json:"query" form:"query"`
		Page     int    `json:"page" form:"page"`
		PageSize int    `json:"page_size" form:"page_size"`

		CustomConfig *SearchCustomConfig `json:"custom_config,omitempty" form:"custom_config,omitempty"`
	}

	SearchCustomConfig struct {
		Policy string `json:"policy"`
		Arch   string `json:"arch"`
	}

	RemoteMetaRequest struct {
		Type                    int
		Uid                     string `json:"uid"`
		RepositorySearchRequest RemoteSearchRequest
		TagSearchRequest        RemoteTagSearchRequest
		TagInfoSearchRequest    RemoteTagInfoSearchRequest
	}

	CallMetaRequest struct {
		Type int    `json:"type"` // 操作类型，必填
		Uid  string `json:"uid"`

		CallGithubRequest        *CallGithubRequest        `json:"callGithubRequest,omitempty"`
		CallKubernetesTagRequest *CallKubernetesTagRequest `json:"callKubernetesTagRequest,omitempty"`
		CallSearchRequest        *CallSearchRequest        `json:"callSearchRequest,omitempty"`
	}

	CreateTaskMessageRequest struct {
		Id      int64  `json:"id"`
		Message string `json:"message"`
	}

	CreateBuildMessageRequest struct {
		Id      int64  `json:"id"`
		Message string `json:"message"`
	}

	CreateSubscribeRequest struct {
		UserMetaRequest `json:",inline"`

		Path       string        `json:"path"`   // 默认会自动填充命名空间，比如 nginx 表示 library/nginx， jenkins/jenkins 则直接使用
		Enable     bool          `json:"enable"` // 启动或者关闭
		Size       int           `json:"size"`   // 同步最新多少个版本, 最多 100 个，普通用户 5
		RegisterId int64         `json:"register_id"`
		Namespace  string        `json:"namespace"`
		Interval   time.Duration `json:"interval"`   // 间隔多久同步一次
		ImageFrom  string        `json:"image_from"` // 镜像来源，支持 dockerhub, gcr, quay.io
		Policy     string        `json:"policy"`     // 默认定义所有版本镜像，支持正则表达式，比如 v1.*
		Arch       string        `json:"arch"`       // 支持的架构，默认不限制  linux/amd64
		Rewrite    bool          `json:"rewrite"`    // 是否覆盖推送
	}

	UpdateSubscribeRequest struct {
		Id              int64 `json:"id"`
		ResourceVersion int64 `json:"resource_version"`

		UserMetaRequest `json:",inline"`
		Enable          bool          `json:"enable"`     // 启动或者关闭
		Size            int           `json:"size"`       // 同步最新多少个版本
		Interval        time.Duration `json:"interval"`   // 间隔多久同步一次
		ImageFrom       string        `json:"image_from"` // 镜像来源，支持 dockerhub, gcr, quay.io
		Policy          string        `json:"policy"`     // 默认定义所有版本镜像，支持正则表达式，比如 v1.*
		Arch            string        `json:"arch"`       // 支持的架构，默认不限制  linux/amd64
		Rewrite         bool          `json:"rewrite"`    // 是否覆盖推送
		Namespace       string        `json:"namespace"`
	}

	RunSubscribeRequest struct {
		UserMetaRequest `json:",inline"`

		SubscribeId int64 `json:"subscribe_id"`
	}

	EnableChartRepoRequest struct {
		UserId      string `json:"user_id,omitempty"`
		UserName    string `json:"user_name,omitempty"`
		ProjectName string `json:"project_name,omitempty"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		Public      bool   `json:"is_public,omitempty"`
	}
)

func (o *CallSearchRequest) SetDefaultPageOption() {
	// 初始化分页属性
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.PageSize <= 0 {
		o.PageSize = 10
	}
	if o.PageSize > 100 {
		o.PageSize = 100
	}
}

func (o *CallSearchRequest) SetNamespace() {
	if len(o.Namespace) == 0 {
		o.Namespace = DefaultDockerhubNamespace
	}
}

// ListOptions is the query options to a standard REST list call.
type ListOptions struct {
	CustomMeta `json:",inline"`

	UserMeta `json:",inline"`
	TaskMeta `json:",inline"`

	PageRequest `json:",inline"` // 分页请求属性
	QueryOption `json:",inline"` // 搜索内容
}

func (o *ListOptions) SetDefaultPageOption() {
	// 初始化分页属性
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.Limit <= 0 {
		o.Limit = 10
	}
	if o.Limit > 100 {
		o.Limit = 100
	}
}

type PageResult struct {
	PageRequest `json:",inline"`

	Total   int64       `json:"total"`   // 总记录数
	Items   interface{} `json:"items"`   // 数据列表
	Message string      `json:"message"` // 正常或异常信息
}

// LoginRequest 账号密码登录请求
type LoginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	RoleStatus  bool   `json:"role_status"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Id          int64  `json:"id" binding:"required"`
	Description string `json:"description"`
	RoleStatus  *bool  `json:"role_status"`
}
