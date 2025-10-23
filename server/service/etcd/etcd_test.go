package etcd

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ⚡ 启动 etcd docker：
// docker run -d --name etcd-test -p 2379:2379 quay.io/coreos/etcd:v3.5.15 /usr/local/bin/etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379

func TestEtcdRegistry(t *testing.T) {
	reg, err := NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 1️⃣ 注册服务
	key := "/game/server/1001"
	value := "127.0.0.1:8001"
	leaseID, err := reg.RegisterService(ctx, key, value, 5)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("✅ Registered key=%s value=%s lease=%v", key, value, leaseID)

	// 2️⃣ 获取列表
	list, err := reg.ListPrefix(ctx, "/game/server/")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("📋 ListPrefix: %v\n", list)

	// 3️⃣ 启动 Watch
	watchCh := make(chan clientv3.Event, 10)
	cancel, err := reg.WatchPrefix(ctx, "/game/server/", watchCh)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	go func() {
		for ev := range watchCh {
			fmt.Printf("🔔 Watch event: %s %s -> %s\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}()

	// 4️⃣ 模拟更新和删除
	time.Sleep(time.Second)
	reg.client.Put(ctx, key, "127.0.0.1:9999")
	time.Sleep(time.Second)
	reg.client.Delete(ctx, key)

	time.Sleep(2 * time.Second)
	log.Println("✅ Test done")
}
