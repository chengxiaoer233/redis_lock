package lock

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"time"
)

// 说明：v1 分布式锁,简单的通过setNx和del来实现
// 问题：（1）相互可以解锁，可以释放别人的锁
//		（2）缺少过期续期机制

type Client struct {

	// 这里定义client为redis.Cmdable的原因是为了兼容
	// redis.Client 和 redis.ClusterClient,可以更通用
	client redis.Cmdable
}

func NewClient(c redis.Cmdable) *Client {
	return &Client{
		client: c,
	}
}

// 获取分布式锁
// 返回值：ok 为 true 加锁成功、false 加锁失败
//       err 不为空则为其他错误，根据实际返回为准
func (c *Client) V1Lock(ctx context.Context, key string, val string, timeout time.Duration) (ok bool, err error) {

	ok, err = c.client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=", err)
		return ok, err
	}

	return
}

// 释放分布式锁
// 返回值：val 为1表示删除成功，val表示删除key的数目，为0时则可能key已经被别人删除，或者已经过期了
//       err 不为空则为其他错误，根据实际返回为准
func (c *Client) V1Unlock(ctx context.Context, key string) (val int64, err error) {

	val, err = c.client.Del(ctx, key).Result()
	if err != nil {
		fmt.Println("解锁失败，err=", err)
		return
	}

	return
}
