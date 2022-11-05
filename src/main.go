package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"redis_lock/dao"
	lock "redis_lock/server"
	"time"
)

func main() {

	ctx := context.Background()
	for i := 0; i < 5; i++ {

		go func() {
			V2LockTest(ctx)
		}()
	}

	time.Sleep(1 * time.Hour)
}

// v1 最简单的加锁和解锁
func V1LockTest(ctx context.Context) {

	// 生成一个redis的client
	c := new(dao.Redis)
	redisClient := c.NewClient()

	// 生成一个 Cmdable client，这里也是可以传入redis.ClusterClient的
	cClient := lock.NewClient(redisClient)

	key := "test"
	val := "val"
	timeout := time.Second * 60

	// 加锁
	ok, err := cClient.V1Lock(ctx, key, val, timeout)
	if err != nil {
		fmt.Println("lock.Lock 加锁失败，err=", err)
		return
	}

	if !ok {
		fmt.Println("分布式锁已经被占用")
		return
	}

	// 模拟业务功能
	fmt.Println("分布式锁加锁成功")
	time.Sleep(time.Second * 1)

	// 释放分布式锁
	res, err := cClient.V1Unlock(ctx, key)
	if err != nil {
		fmt.Println("lock.Lock 解锁失败，err=", err)
		return
	}

	if res != 1 {
		fmt.Println("分布式锁解锁失败，err=", err)
		return
	}

	fmt.Println("分布式锁解锁成功")
}

// v2 谁加锁谁解锁，解锁函数封装在加锁函数返回的实体中
func V2LockTest(ctx context.Context) {

	// 生成一个redis的client
	c := new(dao.Redis)
	redisClient := c.NewClient()

	// 生成一个 Cmdable client，这里也是可以传入redis.ClusterClient的
	cClient := lock.NewClient(redisClient)

	key := "test"
	// vale这里不能再随意设置，需要为uuid,后面删除的时候，需要对比此值是否一致，是则可以删除，否则不行
	val := uuid.New().String()
	timeout := time.Second * 60

	// 加锁,会返回一个lock实体和加锁函数对应的日志
	lock,ok, err := cClient.V2Lock(ctx, key, val, timeout)
	if err != nil {
		fmt.Println("cClient.V2Lock err=", err)
		return
	}

	if !ok {
		fmt.Println("分布式锁已经被占用")
		return
	}

	// 模拟业务功能
	fmt.Println("分布式锁加锁成功")
	time.Sleep(time.Second * 30)

	// 释放分布式锁,通过lock返回的实体才可以删除，不能随意调用
	res, err := lock.V2Unlock(ctx, key)
	if err != nil {
		fmt.Println("lock.Lock 解锁失败，err=", err)
		return
	}

	if res != 1 {
		fmt.Println("分布式锁解锁失败，err=", err)
		return
	}

	fmt.Println("分布式锁解锁成功")
}