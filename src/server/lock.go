package lock

import (
	"context"
	"github.com/go-redis/redis/v9"
	"redis_lock/dao"
	"fmt"
	"time"
)

var client *redis.Client

func init () {
	c := new(dao.Redis)
	client = c.NewClient()
}

// 最简单的加锁和解锁
func Lock(ctx context.Context, key string, val string, timeout time.Duration)(ok bool, err error){

	ok ,err = client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=",err)
		return ok, err
	}

	return
}

func Unlock(ctx context.Context,key string)(val int64, err error){

	val ,err = client.Del(ctx,key).Result()
	if err != nil {
		fmt.Println("解锁失败，err=",err)
		return
	}

	return
}
