package lock

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"time"
)

// 说明：v4 分布式锁,在v4的基础上引入lua脚本，来实现自动续约和手动续约

// 获取分布式锁，设置的key为一个uuid，不是随意值
// 获取成功后，返回的一个lock实体
func (c *Client) V4Lock(ctx context.Context, key string, val string, timeout time.Duration) (lock *Lock, ok bool, err error) {

	ok, err = c.client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=", err)
		return lock, ok, err
	}

	return &Lock{
			client: c.client,
			key:    key,
			val:    val,
		},
		ok, err
}

// 释放分布式锁
// 需要用加锁返回的实体才可以调用
func (l *Lock) V4Unlock(ctx context.Context, luaUnlock string) (int64, error) {

	// 执行lua脚本，原子操作，根据返回的值做不同的处理逻辑
	val, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.val).Int64()
	if err != nil {
		fmt.Println("l.client.Eval err=", err)
		return val, err
	}

	// val返回0时表示key不存在或者key对应的val值不对
	return val, err
}

// 手动续约，执行refresh lua脚本
// 本质就是针对某个key重新设置过期时间
func (l *Lock) refresh(ctx context.Context, luaRefresh string) error {

	res, err := l.client.Eval(ctx, luaRefresh, []string{l.key}, l.val, l.expiration.Seconds()).Int64()
	if err == redis.Nil {
		return errors.New("key 不存在，没有拿到锁")
	}

	if err != nil { // key存在，但是发生了其他服务器错误
		return err
	}

	if res != 1 { // 等于0，锁不是自己的
		return errors.New("不是自己的锁")
	}

	return nil
}
