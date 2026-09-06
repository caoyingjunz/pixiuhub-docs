package rainbow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"k8s.io/klog/v2"

	pb "github.com/caoyingjunz/rainbow/api/rpc/proto"
	"github.com/caoyingjunz/rainbow/pkg/db"
	"github.com/caoyingjunz/rainbow/pkg/types"
)

var (
	RpcClients map[string]pb.Tunnel_ConnectServer
)

// Connect 提供 rpc 注册接口
func (s *ServerController) Connect(stream pb.Tunnel_ConnectServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			klog.Errorf("stream.Recv %v", err)
			return err
		}

		s.lock.Lock()
		if RpcClients == nil {
			RpcClients = make(map[string]pb.Tunnel_ConnectServer)
		}
		old, ok := RpcClients[req.ClientId]
		if !ok || old != stream {
			RpcClients[req.ClientId] = stream
			klog.Infof("client(%s) rpc 注册成功", req.ClientId)
		}
		s.lock.Unlock()
		klog.V(2).Infof("Received %s from %s", string(req.Payload), req.ClientId)
	}
}

func (s *ServerController) preRemoteSearch(ctx context.Context, req types.RemoteSearchRequest) error {
	switch req.Hub {
	case types.ImageHubDocker, types.ImageHubGCR, types.ImageHubQuay, types.ImageHubAll:
	default:
		return fmt.Errorf("unsupported image hub type %s", req.Hub)
	}

	return nil
}

// 兼容前端缺陷
func (s *ServerController) setSearchHubType(req *types.RemoteSearchRequest) {
	// 设置默认仓库类型
	if len(req.Hub) == 0 {
		req.Hub = types.ImageHubDocker
	}
	if req.Hub == "gcr" {
		req.Hub = types.ImageHubGCR
	}
	if req.Hub == "quay" {
		req.Hub = types.ImageHubQuay
	}
}

// 兼容前端缺陷
func (s *ServerController) setRepoHubType(req *types.CallSearchRequest) {
	// 设置默认仓库类型
	if len(req.Hub) == 0 {
		req.Hub = types.ImageHubDocker
	}
	if req.Hub == "gcr" {
		req.Hub = types.ImageHubGCR
	}
	if req.Hub == "quay" {
		req.Hub = types.ImageHubQuay
	}
}

func (s *ServerController) SearchRepositories(ctx context.Context, req types.CallSearchRequest) (interface{}, error) {
	req.Query = strings.TrimSpace(req.Query)
	if len(req.Query) == 0 {
		return []types.CommonSearchRepositoryResult{}, nil
	}
	s.setRepoHubType(&req)
	req.SetNamespace()
	req.SetDefaultPageOption()

	req.TargetType = types.SearchTypeRepo
	key := uuid.NewString()
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallSearchType,
		Uid:               key,
		CallSearchRequest: &req,
	})
	if err != nil {
		klog.Errorf("序列化(%v)失败 %v", req, err)
		return nil, err
	}

	val, err := s.CallV2(ctx, req.ClientId, key, data)
	if err != nil {
		return nil, err
	}

	var searchResp []types.CommonSearchRepositoryResult
	if err = json.Unmarshal(val, &searchResp); err != nil {
		klog.Errorf("序列化 HubSearchResponse失败: %v", err)
		return nil, fmt.Errorf("序列化 HubSearchResponse失败: %v", err)
	}

	return searchResp, nil
}

func (s *ServerController) SearchRepositoryTags(ctx context.Context, req types.CallSearchRequest) (interface{}, error) {
	s.setRepoHubType(&req)
	req.SetDefaultPageOption()
	req.SetNamespace()

	req.TargetType = types.SearchTypeTag
	key := uuid.NewString()
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallSearchType,
		Uid:               key,
		CallSearchRequest: &req,
	})
	if err != nil {
		klog.Errorf("序列化(%v)失败 %v", req, err)
		return nil, err
	}

	val, err := s.CallV2(ctx, req.ClientId, key, data)
	if err != nil {
		return nil, err
	}
	var tagResp types.CommonSearchTagResult
	if err = json.Unmarshal(val, &tagResp); err != nil {
		return nil, err
	}

	return tagResp, nil
}

func (s *ServerController) GetRepositoryTagInfo(ctx context.Context, req types.CallSearchRequest) (interface{}, error) {
	s.setRepoHubType(&req)
	req.SetNamespace()

	key := uuid.NewString()
	req.TargetType = types.SearchTypeTagInfo
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallSearchType,
		Uid:               key,
		CallSearchRequest: &req,
	})
	if err != nil {
		klog.Errorf("序列化(%v)失败 %v", req, err)
		return nil, err
	}

	val, err := s.CallV2(ctx, req.ClientId, key, data)
	if err != nil {
		return nil, err
	}
	var infoResp types.CommonSearchTagInfoResult
	if err = json.Unmarshal(val, &infoResp); err != nil {
		return nil, err
	}
	return infoResp, nil
}

func (s *ServerController) GetRepository(ctx context.Context, req types.CallSearchRequest) (*types.GetRepositoryResult, error) {
	s.setRepoHubType(&req)
	req.SetNamespace()

	key := uuid.NewString()
	req.TargetType = types.GetTypeRepo
	data, err := json.Marshal(types.CallMetaRequest{
		Type:              types.CallSearchType,
		Uid:               key,
		CallSearchRequest: &req,
	})
	if err != nil {
		klog.Errorf("序列化(%v)失败 %v", req, err)
		return nil, err
	}

	val, err := s.CallV2(ctx, req.ClientId, key, data)
	if err != nil {
		return nil, err
	}
	var resp types.GetRepositoryResult
	if err = json.Unmarshal(val, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ServerController) sendMessage(ctx context.Context, clientId string, data []byte) error {
	tags := "all"
	if len(clientId) != 0 {
		tags = clientId
	}
	msg := &primitive.Message{
		Topic: s.cfg.Rocketmq.Topic,
		Body:  data,
	}

	msg.WithTag(tags)
	msg.WithKeys([]string{"PixiuHub"})
	res, err := s.Producer.SendSync(ctx, msg)
	if err != nil {
		klog.Errorf("send message error: %v", err)
		return err
	}

	klog.V(0).Infof("send message success: result=%s", res.String())
	return nil
}

func (s *ServerController) doSearch(ctx context.Context, clientId string, key string, data []byte) ([]byte, error) {
	if err := s.sendMessage(ctx, clientId, data); err != nil {
		return nil, err
	}

	val, err := s.GetResult(ctx, key)
	if err != nil {
		return nil, err
	}
	var sr types.SearchResult
	if err = json.Unmarshal([]byte(val), &sr); err != nil {
		klog.Errorf("反序列化（%v）失败 %v", val, err)
		return nil, err
	}
	if sr.StatusCode != 0 {
		klog.Errorf("doSearch 远程调用失败 %v", err)
		return nil, fmt.Errorf("%s", sr.ErrMessage)
	}

	return sr.Result, nil
}

func (s *ServerController) getActiveNodeName(ctx context.Context) (string, error) {
	agents, err := s.factory.Agent().List(ctx, db.WithStatus("在线"))
	if err != nil {
		return "", err
	}
	if len(agents) == 0 {
		return "", fmt.Errorf("no agent found")
	}

	index := rand.Intn(len(agents))
	return agents[index].Name, nil
}

// CallV2 使用 redis 作为中间件
func (s *ServerController) CallV2(ctx context.Context, clientId string, key string, data []byte) ([]byte, error) {
	// 填充 clientId
	if len(clientId) == 0 {
		var err error
		clientId, err = s.getActiveNodeName(ctx)
		if err != nil {
			return nil, err
		}
		klog.V(0).Infof("agent(%s) selected", clientId)
	}

	if err := s.sendMessageV2(ctx, clientId, key, data); err != nil {
		return nil, err
	}

	val, err := s.GetResult(ctx, key)
	if err != nil {
		return nil, err
	}
	var sr types.CallResult
	if err = json.Unmarshal([]byte(val), &sr); err != nil {
		klog.Errorf("反序列化（%v）失败 %v", val, err)
		return nil, err
	}
	if sr.StatusCode != 0 {
		klog.Errorf("远程调用失败 %v", err)
		return nil, fmt.Errorf("%s", sr.ErrMessage)
	}

	return sr.Result, nil
}

func (s *ServerController) sendMessageV2(ctx context.Context, clientId string, key string, data []byte) error {
	reqKey := fmt.Sprintf("req-%s", key)

	if _, err := s.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, reqKey, data, 30*time.Second)
		pipe.Publish(ctx, fmt.Sprintf("__nodekey@0__:%s", clientId), reqKey)
		return nil
	}); err != nil {
		klog.Errorf("sendMessageV2 临时存储失败 %v", err)
		return err
	}

	klog.V(0).Infof("save req cache to redis success, clientId=%s, key=%s", clientId, reqKey)
	return nil
}

// Call 使用 rocketmq 作为中间件
func (s *ServerController) Call(ctx context.Context, clientId string, key string, data []byte) ([]byte, error) {
	if err := s.sendMessage(ctx, clientId, data); err != nil {
		return nil, err
	}

	val, err := s.GetResult(ctx, key)
	if err != nil {
		return nil, err
	}
	var sr types.CallResult
	if err = json.Unmarshal([]byte(val), &sr); err != nil {
		klog.Errorf("反序列化（%v）失败 %v", val, err)
		return nil, err
	}
	if sr.StatusCode != 0 {
		klog.Errorf("远程调用失败 %v", err)
		return nil, fmt.Errorf("%s", sr.ErrMessage)
	}

	return sr.Result, nil
}

func (s *ServerController) GetResult(ctx context.Context, key string) (string, error) {
	// 先尝试直接获取
	val, err := s.redisClient.Get(ctx, key).Result()
	if err == nil {
		return val, nil // key 存在直接返回
	}
	if err != redis.Nil {
		return "", fmt.Errorf("redis error: %w", err) // 非"不存在"错误
	}

	// key 不存在，准备订阅通知
	channel := fmt.Sprintf("__keyspace@0__:%s", key) // Redis 通知频道格式
	pubSub := s.redisClient.Subscribe(ctx, channel)
	defer pubSub.Close()

	if _, err = pubSub.Receive(ctx); err != nil {
		return "", fmt.Errorf("subscribe failed: %w", err)
	}

	// 再次检查（避免订阅期间 key 被设置）
	val, err = s.redisClient.Get(ctx, key).Result()
	if err == nil {
		return val, nil
	}

	// 60 秒超时
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ch := pubSub.Channel()
	for {
		select {
		case msg := <-ch:
			if msg.Payload == "set" { // 只响应 set 操作
				val, err := s.redisClient.Get(ctx, key).Result()
				if err == nil {
					return val, nil
				}
			}
		case <-waitCtx.Done():
			return "", fmt.Errorf("wait timeout for call(%s)", key)
		}
	}
}
