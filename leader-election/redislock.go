package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	leaderelection "k8s.io/client-go/tools/leaderelection"

	"github.com/ashwinyue/kubernetes-examples/leader-election/redislock"
)

func main() {
	var (
		id            string
		redisAddr     string
		redisUser     string
		redisPass     string
		redisDB       int
		useTLS        bool
		lockKey       string
		leaseSeconds  int
		renewDeadline int
		retryPeriod   int
	)

	flag.StringVar(&id, "id", hostnameOrRand(), "instance identity")
	flag.StringVar(&redisAddr, "redis", "localhost:6379", "redis address or URL (e.g. redis://default:pass@localhost:6379/0)")
	flag.StringVar(&redisUser, "redis-user", "", "redis username (Redis 6+ ACL)")
	flag.StringVar(&redisPass, "redis-pass", "", "redis password")
	flag.IntVar(&redisDB, "redis-db", 0, "redis db index")
	flag.BoolVar(&useTLS, "redis-tls", false, "enable TLS (ignored if rediss:// URL)")
	flag.StringVar(&lockKey, "lock-key", "leader-election:demo", "redis key for leader election")
	flag.IntVar(&leaseSeconds, "lease", 15, "lease duration seconds")
	flag.IntVar(&renewDeadline, "renew", 10, "renew deadline seconds")
	flag.IntVar(&retryPeriod, "retry", 2, "retry period seconds")
	flag.Parse()

	// 构建 Redis 连接配置：优先支持 URL，其次走手动参数
	var rdb *redis.Client
	if strings.HasPrefix(redisAddr, "redis://") {
		opt, err := redis.ParseURL(redisAddr)
		if err != nil {
			log.Fatalf("invalid redis URL: %v", err)
		}
		// URL 优先，命令行可覆盖用户名/密码/DB
		if redisUser != "" {
			opt.Username = redisUser
		}
		if redisPass != "" {
			opt.Password = redisPass
		}
		if redisDB != 0 {
			opt.DB = redisDB
		}
		// rediss:// 自动启用 TLS；如果用 redis:// 但强制 TLS，可以在下方再覆盖
		if useTLS && opt.TLSConfig == nil {
			opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		rdb = redis.NewClient(opt)
	} else {
		opt := &redis.Options{
			Addr:     redisAddr,
			Username: redisUser,
			Password: redisPass,
			DB:       redisDB,
		}
		if useTLS {
			opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		rdb = redis.NewClient(opt)
	}

	// 健康检查（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping err: %v", err)
	}
	log.Printf("connected to redis: %s", safeRedisAddr(redisAddr))

	// 自定义 Redis 锁
	lock := redislock.NewRedisLock(rdb, lockKey, id)

	// 构造选举配置
	cfg := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   time.Duration(leaseSeconds) * time.Second,
		RenewDeadline:   time.Duration(renewDeadline) * time.Second,
		RetryPeriod:     time.Duration(retryPeriod) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Printf("[LEADER] %s started leading", id)
				t := time.NewTicker(2 * time.Second)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						log.Printf("[LEADER] %s context canceled", id)
						return
					case tm := <-t.C:
						log.Printf("[LEADER] %s tick at %s", id, tm.Format(time.RFC3339))
					}
				}
			},
			OnStoppedLeading: func() {
				log.Printf("[FOLLOWER] %s stopped leading", id)
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					log.Printf("[INFO] %s is the new leader", id)
				} else {
					log.Printf("[INFO] current leader elected: %s", identity)
				}
			},
		},
	}

	// 运行选举（阻塞直到上下文取消）
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	elector, err := leaderelection.NewLeaderElector(cfg)
	if err != nil {
		log.Fatalf("create elector err: %v", err)
	}

	go func() {
		time.Sleep(2 * time.Minute)
		log.Printf("stopping after 2 minutes")
		runCancel()
	}()

	elector.Run(runCtx)
	log.Printf("exit")
}

func hostnameOrRand() string {
	h, err := os.Hostname()
	if err == nil && h != "" {
		return h
	}
	return fmt.Sprintf("inst-%d", time.Now().UnixNano())
}

// 打印地址时避免泄露密码
func safeRedisAddr(addr string) string {
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		// 粗略脱敏
		return "[redis URL masked]"
	}
	return addr
}
