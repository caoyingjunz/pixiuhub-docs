package rainbow

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"strings"

	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/db/model"
	"github.com/caoyingjunz/rainbow/pkg/types"
	"github.com/caoyingjunz/rainbow/pkg/util/docker"
)

func (s *ServerController) CreateRegistry(ctx context.Context, req *types.CreateRegistryRequest) error {
	if strings.Contains(req.Password, ";") {
		return fmt.Errorf("镜像仓库密码不能包含特殊字符串 !")
	}

	_, err := s.factory.Registry().Create(ctx, &model.Registry{
		Name:       req.Name,
		UserId:     req.UserId,
		Repository: req.Repository,
		Namespace:  req.Namespace,
		Username:   req.Username,
		Password:   req.Password,
		Role:       req.Role,
	})

	return err
}

func (s *ServerController) LoginRegistry(ctx context.Context, req *types.CreateRegistryRequest) error {
	if err := docker.LoginDocker(req.Repository, req.Username, req.Password); err != nil {
		klog.Errorf("登陆镜像仓库 (%s) 失败 %v", req.Repository, err)
		return err
	}

	if err := docker.LogoutDocker(req.Repository); err != nil {
		klog.Warningf("退出 %s 异常 %v", req.Repository, err)
	}
	return nil
}

func (s *ServerController) UpdateRegistry(ctx context.Context, req *types.UpdateRegistryRequest) error {
	if strings.Contains(req.Password, ";") {
		return fmt.Errorf("镜像仓库密码不能包含特殊字符串 !")
	}

	return s.factory.Registry().Update(ctx, req.Id, req.ResourceVersion, map[string]interface{}{
		"user_id":    req.UserId,
		"repository": req.Repository,
		"namespace":  req.Namespace,
		"username":   req.Username,
		"password":   req.Password,
	})
}

func (s *ServerController) DeleteRegistry(ctx context.Context, registryId int64) error {
	return s.factory.Registry().Delete(ctx, registryId)
}

func (s *ServerController) ListRegistries(ctx context.Context, listOption types.ListOptions) (interface{}, error) {
	return s.factory.Registry().List(ctx, db.WithUser(listOption.UserId), db.WithNameLike(listOption.NameSelector))
}

func (s *ServerController) GetRegistry(ctx context.Context, registryId int64) (interface{}, error) {
	return s.factory.Registry().Get(ctx, registryId)
}
