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
func Lock(ctx context.Context, key string, val string, timeout time.Duration)(err error){

	ok ,err := client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=",err)
		return err
	}

	if ok{
		fmt.Println("加分布式锁成功")
	}else{
		fmt.Println("加分布式锁失败")
	}

	return
}

func Unlock(ctx context.Context,key string)(err error){

	val ,err := client.Del(ctx,key).Result()
	if err != nil {
		fmt.Println("解锁失败，err=",err)
		return
	}

	if val == 1{
		fmt.Println("解锁成功")
	}else{
		fmt.Println("解锁失败")
	}

	return
}
