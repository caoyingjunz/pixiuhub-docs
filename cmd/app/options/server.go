package options

import (
	"fmt"
	"os"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"github.com/caoyingjunz/pixiulib/config"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/goharbor/go-client/pkg/harbor"
	v2client "github.com/goharbor/go-client/pkg/sdk/v2.0/client"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	rainbowconfig "github.com/caoyingjunz/rainbow/cmd/app/config"
	"github.com/caoyingjunz/rainbow/pkg/controller"
	rainbowdb "github.com/caoyingjunz/rainbow/pkg/db"
)

type ServerOptions struct {
	ComponentConfig rainbowconfig.Config
	ConfigFile      string

	db      *gorm.DB
	Factory rainbowdb.ShareDaoFactory

	RedisClient *redis.Client
	Producer    rocketmq.Producer
	ChartAPI    *v2client.HarborAPI

	HttpEngine *gin.Engine
	Controller controller.RainbowInterface
}

func NewServerOptions(configFile string) (*ServerOptions, error) {
	return &ServerOptions{
		HttpEngine: gin.Default(),
		ConfigFile: configFile,
	}, nil
}

// GetDB 返回已初始化的 gorm 数据库实例（供审计中间件等使用）
func (o *ServerOptions) GetDB() *gorm.DB {
	return o.db
}

// Complete completes all the required options
func (o *ServerOptions) Complete() error {
	// 配置文件优先级: 默认配置，环境变量，命令行
	if len(o.ConfigFile) == 0 {
		// Try to read config file path from env.
		if cfgFile := os.Getenv("ConfigFile"); cfgFile != "" {
			o.ConfigFile = cfgFile
		} else {
			o.ConfigFile = defaultConfigFile
		}
	}

	c := config.New()
	c.SetConfigFile(o.ConfigFile)
	c.SetConfigType("yaml")

	if err := c.Binding(&o.ComponentConfig); err != nil {
		klog.Fatal(err)
	}

	// 设置配置默认值
	o.ComponentConfig.SetDefaults()

	// 启动前校验关键配置（避免以空凭据等弱配置静默运行）
	if err := o.ComponentConfig.Valid(); err != nil {
		return err
	}

	// 注册依赖组件
	if err := o.register(); err != nil {
		return err
	}

	// 首次启动引导：用户表为空时创建初始管理员账号
	if err := o.bootstrap(); err != nil {
		return err
	}
	// 注册 redis 客户端
	if err := o.registerRedis(); err != nil {
		return err
	}
	//if err := o.registerProducer(); err != nil {
	//	return err
	//}

	// 注册 chart repo 客户端
	if err := o.registerChartAPI(); err != nil {
		return err
	}

	o.Controller = controller.New(o.ComponentConfig, o.Factory, o.RedisClient, o.Producer, o.ChartAPI)
	return nil
}

func (o *ServerOptions) registerProducer() error {
	rlog.SetLogLevel("warn")
	rocketmqConfig := o.ComponentConfig.Rocketmq
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(rocketmqConfig.NameServers), // NameServer地址
		producer.WithRetry(3),                               // 重试次数
		producer.WithGroupName(rocketmqConfig.GroupName),    // 生产者组名
		producer.WithCredentials(primitive.Credentials{AccessKey: rocketmqConfig.Credential.AccessKey, SecretKey: rocketmqConfig.Credential.SecretKey}),
	)
	if err != nil {
		return err
	}

	o.Producer = p
	return nil
}

func (o *ServerOptions) register() error {
	sqlConfig := o.ComponentConfig.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		sqlConfig.User,
		sqlConfig.Password,
		sqlConfig.Host,
		sqlConfig.Port,
		sqlConfig.Name)

	opt := &gorm.Config{}
	db, err := gorm.Open(mysql.Open(dsn), opt)
	if err != nil {
		return err
	}
	o.db = db

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)

	o.Factory, err = rainbowdb.NewDaoFactory(db, true)
	return err
}

func (o *ServerOptions) registerRedis() error {
	redisConfig := o.ComponentConfig.Redis
	o.RedisClient = redis.NewClient(&redis.Options{
		Addr:        redisConfig.Addr,
		Username:    redisConfig.Username,
		Password:    redisConfig.Password,
		DB:          redisConfig.Db,
		ReadTimeout: 10 * time.Second,
	})

	return nil
}

func (o *ServerOptions) registerChartAPI() error {
	harborCfg := o.ComponentConfig.Server.Harbor
	cs, err := harbor.NewClientSet(&harbor.ClientSetConfig{
		URL:      harborCfg.URL,
		Username: harborCfg.Username,
		Password: harborCfg.Password,
	})
	if err != nil {
		return err
	}
	o.ChartAPI = cs.V2()
	return nil
}
