package config

import (
	"fmt"
	"os"
)

const (
	DefaultNormalRateLimitMaxRequests  = 100
	DefaultSpecialRateLimitMaxRequests = 50
	DefaultSpecialRateLimitedPath      = "/rainbow/search"
	DefaultUserRateLimitCap            = 1000
	DefaultUserRateLimitQuantum        = 10
	DefaultUserRateLimitCapacity       = 100

	defaultRainbowdTemplateDir = "/data/template"
	defaultDownloadDir         = "/data/pixiuctl"
)

// 敏感凭据支持的环境变量名：环境变量优先于 config.yaml（用于 Secret 注入，避免明文凭据入库）
const (
	EnvHarborUsername   = "RAINBOW_HARBOR_USERNAME"
	EnvHarborPassword   = "RAINBOW_HARBOR_PASSWORD"
	EnvMysqlPassword    = "RAINBOW_MYSQL_PASSWORD"
	EnvRedisPassword    = "RAINBOW_REDIS_PASSWORD"
	EnvRegistryUsername = "RAINBOW_REGISTRY_USERNAME"
	EnvRegistryPassword = "RAINBOW_REGISTRY_PASSWORD"
	EnvRocketmqAK       = "RAINBOW_ROCKETMQ_ACCESS_KEY"
	EnvRocketmqSK       = "RAINBOW_ROCKETMQ_SECRET_KEY"
	EnvAuthAccessKey    = "RAINBOW_AUTH_ACCESS_KEY"
	EnvAuthSecretKey    = "RAINBOW_AUTH_SECRET_KEY"
	EnvJWTKey           = "RAINBOW_JWT_KEY"
	EnvEncryptKey       = "RAINBOW_ENCRYPT_KEY"
)

// SetDefaults 设置配置的默认值
func (c *Config) SetDefaults() {
	if c.RateLimit.NormalRateLimit.MaxRequests == 0 {
		c.RateLimit.NormalRateLimit.MaxRequests = DefaultNormalRateLimitMaxRequests
	}
	if c.RateLimit.SpecialRateLimit.MaxRequests == 0 {
		c.RateLimit.SpecialRateLimit.MaxRequests = DefaultSpecialRateLimitMaxRequests
	}
	if c.RateLimit.UserRateLimit.Cap == 0 {
		c.RateLimit.UserRateLimit.Cap = DefaultUserRateLimitCap
	}
	if c.RateLimit.UserRateLimit.Quantum == 0 {
		c.RateLimit.UserRateLimit.Quantum = DefaultUserRateLimitQuantum
	}
	if c.RateLimit.UserRateLimit.Capacity == 0 {
		c.RateLimit.UserRateLimit.Capacity = DefaultUserRateLimitCapacity
	}
	if c.RateLimit.SpecialRateLimit.RateLimitedPath == nil {
		c.RateLimit.SpecialRateLimit.RateLimitedPath = []string{DefaultSpecialRateLimitedPath}
	}
	if len(c.Server.DownloadDir) == 0 {
		c.Server.DownloadDir = defaultDownloadDir
	}

	// 环境变量覆盖明文凭据（Secret 注入）
	c.overrideCredentialsFromEnv()
}

// overrideCredentialsFromEnv 环境变量优先覆盖 config.yaml 中的敏感凭据
func (c *Config) overrideCredentialsFromEnv() {
	setIfEnv := func(env string, dst *string) {
		if v := os.Getenv(env); len(v) != 0 {
			*dst = v
		}
	}

	setIfEnv(EnvHarborUsername, &c.Server.Harbor.Username)
	setIfEnv(EnvHarborPassword, &c.Server.Harbor.Password)
	setIfEnv(EnvMysqlPassword, &c.Mysql.Password)
	setIfEnv(EnvRedisPassword, &c.Redis.Password)
	setIfEnv(EnvRegistryUsername, &c.Registry.Username)
	setIfEnv(EnvRegistryPassword, &c.Registry.Password)
	setIfEnv(EnvRocketmqAK, &c.Rocketmq.Credential.AccessKey)
	setIfEnv(EnvRocketmqSK, &c.Rocketmq.Credential.SecretKey)
	setIfEnv(EnvAuthAccessKey, &c.Server.Auth.AccessKey)
	setIfEnv(EnvAuthSecretKey, &c.Server.Auth.SecretKey)
	setIfEnv(EnvJWTKey, &c.Server.JWTKey)
	setIfEnv(EnvEncryptKey, &c.Server.EncryptKey)
}

// Valid 启动前校验关键配置，避免以空凭据等弱配置静默运行
func (c *Config) Valid() error {
	if c.Default.Mode != "debug" && c.Default.Mode != "release" {
		return fmt.Errorf("无效的运行模式 mode: %s（仅支持 debug/release）", c.Default.Mode)
	}
	if c.Mysql.Host == "" || c.Mysql.User == "" || c.Mysql.Name == "" {
		return fmt.Errorf("MySQL 配置不完整（host/user/name 不能为空）")
	}
	if c.Server.Auth.AccessKey == "" || c.Server.Auth.SecretKey == "" {
		return fmt.Errorf("认证凭据未配置（建议通过环境变量 %s/%s 注入）", EnvAuthAccessKey, EnvAuthSecretKey)
	}

	return nil
}

type Config struct {
	Default DefaultOption `yaml:"default"`

	Mysql MysqlOptions `yaml:"mysql"`
	Redis RedisOption  `yaml:"redis"`

	Kubernetes KubernetesOption `yaml:"kubernetes"`
	Images     []Image          `yaml:"images"`

	Server   ServerOption   `yaml:"server"`
	Rainbowd RainbowdOption `yaml:"rainbowd"`

	Rocketmq RocketmqOption `yaml:"rocketmq"`

	Plugin   PluginOption `yaml:"plugin"`
	Registry Registry     `yaml:"registry"`

	Build *BuildOption `yaml:"build,omitempty"`

	Agent AgentOption `yaml:"agent"`

	RateLimit RateLimitOption `yaml:"rate_limit"`
}

type DefaultOption struct {
	Listen int    `yaml:"listen"`
	Mode   string `yaml:"mode"` // debug 和 release 模式

	PushKubernetes bool `yaml:"push_kubernetes"`
	PushImages     bool `yaml:"push_images"`

	Time int64 `yaml:"time"`
}

type ServerOption struct {
	DownloadDir string `yaml:"download_dir"`
	Auth        Auth   `yaml:"auth"`
	Harbor      Harbor `yaml:"harbor"`
	JWTKey      string `yaml:"jwt_key"`
	// EncryptKey 用于敏感字段（如 AK/SK SecretKey）加密，通过环境变量注入
	EncryptKey string `yaml:"encrypt_key"`
}

type RainbowdOption struct {
	Name        string     `yaml:"name"`
	TemplateDir string     `yaml:"template_dir"`
	DataDir     string     `yaml:"data_dir"`
	AgentImage  string     `yaml:"agent_image"`
	Nodes       []NodeSpec `yaml:"nodes,omitempty"`
}

type NodeSpec struct {
	Name string `yaml:"name,omitempty"`
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

type RocketmqOption struct {
	NameServers []string   `yaml:"name_servers"`
	GroupName   string     `yaml:"group_name"`
	Topic       string     `yaml:"topic"`
	Credential  Credential `yaml:"credential"`
}

type Credential struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

func (r *RainbowdOption) SetDefault() {
	if len(r.TemplateDir) == 0 {
		r.TemplateDir = defaultRainbowdTemplateDir
	}
}

type Harbor struct {
	URL       string `yaml:"url"`
	Namespace string `yaml:"namespace"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type Auth struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type KubernetesOption struct {
	Version string `yaml:"version"`
}

type PluginOption struct {
	Callback   string `yaml:"callback"`
	TaskId     int64  `yaml:"task_id"`
	RegistryId int64  `yaml:"registry_id"`
	Synced     bool   `yaml:"synced"`
	Driver     string `yaml:"driver"`
	Arch       string `yaml:"arch"`
}

type BuildOption struct {
	Callback       string `yaml:"callback"`
	BuildId        int64  `yaml:"build_id"`
	Arch           string `yaml:"arch"`
	Repo           string `yaml:"repo"`
	DockerfilePath string `yaml:"dockerfile_path"`
}

type Registry struct {
	Repository string `yaml:"repository"`
	Namespace  string `yaml:"namespace"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
}

type MysqlOptions struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
}

type RedisOption struct {
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}

type AgentOption struct {
	Name       string `yaml:"name"`
	DataDir    string `yaml:"data_dir"`
	RetainDays int    `yaml:"retain_days"`
}

type RateLimitOption struct {
	NormalRateLimit  NormalRateLimit  `yaml:"normal_rate_limit"`
	SpecialRateLimit SpecialRateLimit `yaml:"special_rate_limit"`
	UserRateLimit    UserRateLimit    `yaml:"user_rate_limit"`
}

type UserRateLimit struct {
	Cap      int `yaml:"cap"`
	Quantum  int `yaml:"quantum"`
	Capacity int `yaml:"capacity"`
}

type NormalRateLimit struct {
	MaxRequests int `yaml:"max_requests"`
}
type SpecialRateLimit struct {
	RateLimitedPath []string `yaml:"rate_limited_path"`
	MaxRequests     int      `yaml:"max_requests"`
}

type PluginTemplateConfig struct {
	Default    DefaultOption    `yaml:"default"`
	Kubernetes KubernetesOption `yaml:"kubernetes"`
	Plugin     PluginOption     `yaml:"plugin"`
	Registry   Registry         `yaml:"registry"`
	Images     []Image          `yaml:"images"`
}

type Image struct {
	Name string   `yaml:"name"`
	Id   int64    `yaml:"id"`
	Path string   `yaml:"path"`
	Tags []string `yaml:"tags"`
}

func (i Image) GetMap(repo, ns string) map[string]string {
	m := make(map[string]string)
	for _, tag := range i.Tags {
		m[i.Path+":"+tag] = repo + "/" + ns + "/" + i.Name + ":" + tag
	}
	return m
}

func (i Image) GetId() int64 {
	return i.Id
}

func (i Image) GetPath() string {
	return i.Path
}
