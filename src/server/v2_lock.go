package lock

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"time"
)

// 说明：v2 分布式锁,在v1的基础上增加了校验机制，谁加锁谁解锁
// 问题：（1）删除的时候，判断val是否一致和删除的动作不是原子的，可能会删除别人的锁

// 业务加锁时返回的lock实体，主要目的是防止别人删除自己的锁或删除别人的锁
type Lock struct {
	client     redis.Cmdable
	key        string
	val        string
	expiration time.Duration
}

// 获取分布式锁，设置的key为一个uuid，不是随意值
// 获取成功后，返回的一个lock实体
func (c *Client) V2Lock(ctx context.Context, key string, val string, timeout time.Duration) (lock *Lock, ok bool, err error) {

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
func (l *Lock) V2Unlock(ctx context.Context, key string) (int64, error) {

	//先获取key对应的值，这里key可能已经被删除了
	val, err := l.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil { // 其他错误
		fmt.Println("l.client.Get err=", err)
		return 0, err

	} else if err == redis.Nil { // key被删了
		fmt.Println("redis key is nil")
		return 0, errors.New("redis key is nil")
	}

	// 判断获取的值是否和自己的值相等，相等则删除，不等则直接返回
	if l.val == val { //是自己的，可以删除

		val, err := l.client.Del(ctx, key).Result()
		if err != nil {
			fmt.Println("l.client.Del err=", err)
			return 0, err
		}
		return val, err
	}

	// 不是自己的，直接打印错误退出
	fmt.Println("分布式锁已被重新获取，不执行删除")
	return 0, errors.New("分布式锁已被重新获取，不执行删除")
}
