package redislock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// RedisLock 是一个满足 resourcelock.Interface 的自定义实现，
// 使用 Redis 键存储 LeaderElectionRecord，并用 TTL 表达租约。
type RedisLock struct {
	client   *redis.Client
	key      string
	identity string
	// 注意：LeaseDuration 由选举器控制，这里只负责更新 TTL，保持与 record 一致。
}

func NewRedisLock(client *redis.Client, key, identity string) *RedisLock {
	return &RedisLock{
		client:   client,
		key:      key,
		identity: identity,
	}
}

// leaderElectionState 是 Redis 中存储的值（JSON）。
type leaderElectionState struct {
	Record resourcelock.LeaderElectionRecord `json:"record"`
	// 可选：增加一个伪 resourceVersion/epoch，用于更强的 CAS。
	Epoch string `json:"epoch"`
}

func (l *RedisLock) Identity() string { return l.identity }
func (l *RedisLock) Describe() string { return fmt.Sprintf("redis/%s", l.key) }

// RecordEvent 供事件系统使用；这里简单打印日志。
func (l *RedisLock) RecordEvent(s string) { log.Printf("[event] %s", s) }

// Get 返回当前的 LeaderElectionRecord（如果不存在，返回空记录与错误置 nil 以符合常规期望）。
func (l *RedisLock) Get(ctx context.Context) (*resourcelock.LeaderElectionRecord, []byte, error) {
	b, err := l.client.Get(ctx, l.key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 不存在视为无锁
			// GroupResource 中 Resource 使用复数资源名，核心组 group 为空字符串
			gr := schema.GroupResource{Group: "", Resource: "redislocks"}
			return &resourcelock.LeaderElectionRecord{}, nil, apierrors.NewNotFound(gr, l.identity)
		}
		return nil, nil, err
	}
	var st leaderElectionState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, nil, err
	}
	return &st.Record, b, nil
}

// Create 创建租约，要求键不存在（NX 语义）。
func (l *RedisLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	st := leaderElectionState{Record: ler, Epoch: strconv.FormatInt(time.Now().UnixNano(), 10)}
	data, _ := json.Marshal(st)
	ttl := time.Duration(ler.LeaseDurationSeconds) * time.Second

	ok, err := l.client.SetNX(ctx, l.key, data, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		// 与 K8s 语义对齐：已存在时返回冲突错误
		return fmt.Errorf("lock already exists")
	}
	return nil
}

// Update 用于 Leader 续约/保持租约，仅允许当前持有者更新；同时刷新 TTL。
func (l *RedisLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	// 使用 Lua 脚本以原子方式校验持有者并更新
	script := redis.NewScript(`
local key = KEYS[1]
local val = redis.call("GET", key)
if not val then
  return {err="not found"}
end
local current = cjson.decode(val)
if current.record.holderIdentity ~= ARGV[1] and current.record.holderIdentity ~= "" then
  return {err="not holder"}
end
current.record = cjson.decode(ARGV[2])
current.epoch = ARGV[3]
local updated = cjson.encode(current)
redis.call("SET", key, updated, "PX", ARGV[4])
return "OK"
`)
	// st := leaderElectionState{Record: ler, Epoch: time.Now().UnixNano()}
	data, _ := json.Marshal(ler)
	ttl := (time.Duration(ler.LeaseDurationSeconds) * time.Second).Milliseconds()

	_, err := script.Run(ctx, l.client, []string{l.key},
		l.identity, string(data), strconv.FormatInt(time.Now().UnixNano(), 10), fmt.Sprintf("%d", ttl),
	).Result()
	if err != nil {
		return err
	}
	return nil
}
