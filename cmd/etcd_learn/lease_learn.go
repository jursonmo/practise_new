package main

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// 总结：
// testLease1()证明：
// 1. 取消cli.KeepAlive(ctx, lease.ID) 中的ctx, 则会自动取消对应的租约续约,, keeaplive 对应的ch关闭, 但是租约ID的对象在服务器上依然存在, 关联的key 会在租约过期后自动删除
// 2. 跟cli.Revoke(lease.ID)不一样是, Revoke 不但关闭keeaplive 对应的ch, 还会让etcd server 撤销租约的同时会立即删除对应的key。
// 3. 取消对应的租约续约后，租约ID在服务器上依然存在,这个时候可以li.Revoke(lease.ID)来删除租约，不会返回错误, 但是会立即删除key。如果租约到期了，被etcd server删除了, 再调用cli.Revoke(lease.ID) 会返回错误etcdserver: requested lease not found

// testLease2()证明：
// 测试一个key 更换租约后，旧的租约是否会被自动删除,还能否keep alive. 结论是旧租约不会被自动删除，只要一直客户端保持续约，租约就不会被自动删除。
func main() {
	//testLease1()
	//测试一个key 更换租约后，旧的租约是否会被自动删除,还能否keep alive
	testLease2()
}

// 验证一个etcd client 可以同时创建多个租约, 并且可以同时给多个key设置租约. 以及撤销租约的两种方式。
func testLease1() {
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"}, // etcd 服务地址
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// 创建一个租约ID, 底层是一个lease client对象, 这个lease client 可以向服务器申请一个租约ID, 这个租约ID可以用来给key设置一个租约, 当key的租约到期时, etcd server 会自动删除key.
	lease1, err := cli.Grant(context.Background(), 5)
	if err != nil {
		panic(err)
	}

	lease2, err := cli.Grant(context.Background(), 10)
	if err != nil {
		panic(err)
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	_ = cancel1

	SetAndKeepAlive(ctx1, cli, lease1, "key1", "value1")
	SetAndKeepAlive(context.Background(), cli, lease2, "key2", "value2")

	// 模拟业务运行
	time.Sleep(10 * time.Second)
	fmt.Printf("cancel1")
	cancel1() //取消ctx, 则会自动取消对应的租约续约, keeaplive ch关闭, 但是租约在服务器上依然存在, 跟cli.Revoke(lease.ID) 不一样

	time.Sleep(2 * time.Second)
	fmt.Println("revoke lease1")
	_, err = cli.Revoke(context.Background(), lease1.ID)
	if err != nil {
		fmt.Println(err) // 如果租约在服务器上已经过期被删除，则会返回错误etcdserver: requested lease not found
	} else {
		fmt.Println("revoke lease1 ok") //如果只是取消了租约，但是租约在服务器上依然存在, cli.Revoke() ，则会返回nil, 并立即删除key。
	}

	select {}
}

// 测试一个key 更换租约后，旧的租约是否会被自动删除,还能否keep alive. 结论是旧租约不会被自动删除，只要一直客户端保持续约，租约就不会被自动删除。
func testLease2() {
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"}, // etcd 服务地址
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	lease1, err := cli.Grant(context.Background(), 5)
	if err != nil {
		panic(err)
	}

	lease2, err := cli.Grant(context.Background(), 20)
	if err != nil {
		panic(err)
	}
	SetAndKeepAlive(context.Background(), cli, lease1, "key1", "value1")
	time.Sleep(time.Second * 10)
	fmt.Println("--------set key1 with lease2------")
	ctx, cancel := context.WithCancel(context.Background())
	SetAndKeepAlive(ctx, cli, lease2, "key1", "value1") //把key1的租约改为lease2, 则在服务器上lease1的租约和key1自动解除, 但是服务器的lease1 依然存在, .
	time.Sleep(time.Second * 10)
	fmt.Println("--------cancel ctx lease2------") //取消lease2后，key1 会在lease2租约过期后自动删除, 但是服务器的lease1 依然存在, 客户端一直可以保持keepalive 租约lease1
	cancel()
	time.Sleep(time.Second * 25)
	// lease1 已经跟key1 解除了关联.
	// 这个时候，lease1 的keepalive ch 还在打印，说明lease1 还在服务器上存在, 但是key1 已经因为lease2 的到期被删除了
	select {}

}

func SetAndKeepAlive(ctx context.Context, cli *clientv3.Client, lease *clientv3.LeaseGrantResponse, key, value string) {
	_, err := cli.Put(ctx, key, value, clientv3.WithLease(lease.ID))
	if err != nil {
		panic(err)
	}
	// 保持租约（可选，如果需要长期续约）
	// 取消ctx, 则会自动取消续约租约, ch channel 会被关闭, 但是租约在服务器上依然存在, key会在租约过期后自动删除, 跟cli.Revoke(lease.ID)不一样,Revoke  会让etcd server 撤销租约的同时会立即删除对应的key。
	ch, kaErr := cli.KeepAlive(ctx, lease.ID)
	if kaErr != nil {
		fmt.Printf("KeepAlive error: %v\n", kaErr)
		return
	}

	// 打印 KeepAlive 响应
	go func() {
		for ka := range ch {
			fmt.Printf("KeepAlive response: TTL=%d, ka.ID:%d, lease id:%d\n", ka.TTL, ka.ID, lease.ID)
		}
		fmt.Printf("KeepAlive quit, lease id:%d\n", lease.ID)
	}()
}
