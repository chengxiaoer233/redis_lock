package main

import (
	"context"
	"fmt"
	"redis_lock/server"
	"time"
)

func main(){

	ctx := context.Background()
	for i := 0 ; i < 5 ; i++ {

		go func(){
			handle(ctx)
		}()
	}

	time.Sleep(time.Second * 10)
}

func handle(ctx context.Context){

	key := "test"
	val := "val"
	timeout := time.Second * 10

	// 加锁
	ok,err := lock.Lock(ctx,key,val, timeout)
	if err != nil {
		fmt.Println("lock.Lock 加锁失败，err=",err)
		return
	}

	if !ok{
		fmt.Println("分布式锁已经被占用")
		return
	}

	// 模拟业务功能
	fmt.Println("分布式锁加锁成功")
	time.Sleep(time.Second * 1)

	// 释放分布式锁
	res, err := lock.Unlock(ctx,key)
	if err != nil {
		fmt.Println("lock.Lock 解锁失败，err=",err)
		return
	}

	if res != 1{
		fmt.Println("分布式锁解锁失败，err=",err)
		return
	}

	fmt.Println("分布式锁解锁成功")
}