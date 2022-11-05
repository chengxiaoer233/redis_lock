<div align="center">

# 基于redis的分布式锁实现
</div>


<div align="center">

![](https://img.shields.io/github/languages/code-size/chengxiaoer233/redis_lock?label=CodeSize)
![](https://img.shields.io/github/stars/chengxiaoer233/redis_lock?label=GitHub)
![](https://img.shields.io/github/watchers/chengxiaoer233/redis_lock?label=Watch)
[![Go Report Card](https://goreportcard.com/badge/github.com/chengxiaoer233/redis_lock)](https://goreportcard.com/report/github.com/chengxiaoer233/redis_lock)
[![LICENSE](https://img.shields.io/badge/license-MIT-green)](https://mit-license.org/)
</div>


<div align="center">

<img  src="https://my-source666.obs.cn-south-1.myhuaweicloud.com/myBlog/golang-jixiangwu-image.png" width="600" height="350"/>

</div>

#### 版本一：最简易的redis分布式

* 结构体定义
```go
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
```

* 获取分布式锁
```go
func (c *Client) V1Lock(ctx context.Context, key string, val string, timeout time.Duration) (ok bool, err error) {

	ok, err = c.client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=", err)
		return ok, err
	}

	return
}
```
* 释放分布式锁
```go
func (c *Client) V1Unlock(ctx context.Context, key string) (val int64, err error) {

	val, err = c.client.Del(ctx, key).Result()
	if err != nil {
		fmt.Println("解锁失败，err=", err)
		return
	}

	return
}
```
main函数
```go
package main

import (
	"context"
	"fmt"
	"redis_lock/dao"
	lock "redis_lock/server"
	"time"
)

func main() {

	ctx := context.Background()
	for i := 0; i < 5; i++ {

		go func() {
			V1LockTest(ctx)
		}()
	}

	time.Sleep(time.Second * 10)
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
	time.Sleep(time.Second * 30)

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
```

* 输出结果
```shell script
分布式锁加锁成功
分布式锁已经被占用
分布式锁已经被占用
分布式锁已经被占用
分布式锁已经被占用
分布式锁解锁成功
```
* 分析  

    分布式锁的本质就是一个Redis键值对  

    **加锁** 
    ```shell script
  
    setnx（SET if Not eXists）
  
    参数讲解：
      key ：健
      value：值
      expiration ：过期时间
  
    指定的 key 不存在时，为 key 设置指定的值，带上超时时间
    
    返回值：
      1 ：设置成功, 0 ：设置失败
    ```
  
    **解锁** 
    ```shell script
  
    DEL命令：用于删除已存在的键。不存在的 key 会被忽略。
  
    参数讲解：
      key ：需要删除的健
    
    返回值：
       被删除 key 的数量
       如果key存在删除成功则返回1，不存在则返回0
    ```
 * 存在问题
    + 1：第一个用户拿到锁，业务处理时间过长，锁超时，被人拿走，该处理完后，删除了不是自己创建的分布式锁
    + 2：没有过期续期机制  
    
    
#### 版本二：uuid来确保用户只能删除自己的锁  

 - 原理：  
  （1）加锁的时候，设置key、value的时候，不能随意设置值，而是设置一个全局唯一的uuid  
  （2）解锁的时候，需要判断当前key对应的value是否是自己设置的value，是则证明是自己加锁的，
 可以释放，不是则证明不是自己加锁的，中间锁已经被人拿去了，不能自己释放
 
 * 结构体定义
 ```shell script
// 业务加锁时返回的lock实体，主要目的是防止别人删除自己的锁或删除别人的锁
// 加锁成功的时候返回

type Lock struct {
	client redis.Cmdable
	key string
	val string
}
```

* 获取分布式锁
    + 设置的key为一个uuid，不是随意值
    + 获取成功后，返回的一个lock实体 
```go
func (c *Client) V2Lock(ctx context.Context, key string, val string, timeout time.Duration) (lock *Lock, ok bool, err error) {
    
    // 获取分布式锁
	ok, err = c.client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=", err)
		return lock, ok, err
	}
    
    // 返回lock实体
	return &Lock{
		client: c.client,
		key: key,
		val: val,
	},
	ok,err
}
```     
* 释放分布式锁
    + 需要用加锁返回的实体才可以调用 
```go
func (l *Lock) V2Unlock(ctx context.Context, key string)(int64, error){

	//先获取key对应的值，这里key可能已经被删除了
	val, err := l.client.Get(ctx,key).Result()
	if err != nil && err != redis.Nil { // 其他错误
		fmt.Println("l.client.Get err=",err)
		return 0,err

	}else if err == redis.Nil{ // key被删了
		fmt.Println("redis key is nil")
		return 0,errors.New("redis key is nil")
	}

	// 判断获取的值是否和自己的值相等，相等则删除，不等则直接返回
	if l.val == val { //是自己的，可以删除

		val, err := l.client.Del(ctx,key).Result()
		if err != nil {
			fmt.Println("l.client.Del err=",err)
			return 0,err
		}
		return val, err
	}

	// 不是自己的，直接打印错误退出
	fmt.Println("分布式锁已被重新获取，不执行删除")
	return 0, errors.New("分布式锁已被重新获取，不执行删除")
}
``` 

* main函数
```go
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
```
* 输出结果
    
    + 正常加解锁
    
    ```shell script
      分布式锁加锁成功
      分布式锁已经被占用
      分布式锁已经被占用
      分布式锁已经被占用
      分布式锁已经被占用
    ```
    
    + 加锁后，通过在终端手动删除key
        
        + 获取锁成功
        
        ```shell script
            分布式锁加锁成功
            分布式锁已经被占用
            分布式锁已经被占用
            分布式锁已经被占用
            分布式锁已经被占用
        ```
        
        + 在redis-cli终端删除key
        
        ```shell script
          127.0.0.1:6379> get test
          "22d59132-d3f9-44c8-abfc-cffa3febd568"
      
          127.0.0.1:6379> del test
          (integer) 1
        ```
        
        + 业务处理结果最终输出
        
        ```shell script
          redis key is nil
          lock.Lock 解锁失败，err= redis key is nil
        ```
    
    + 加锁后，通过在终端手动修改key对应的val值
                
        + 获取锁成功
        
        ```shell script
            分布式锁加锁成功
            分布式锁已经被占用
            分布式锁已经被占用
            分布式锁已经被占用
            分布式锁已经被占用
        ```
        
        + 在redis-cli终端修改key的val
        
        ```shell script
      
          127.0.0.1:6379> get test
          "a86715fd-45d7-4508-8c64-b34fb9b4a914"
      
          127.0.0.1:6379> set test aaaaaaa
          OK
      
          127.0.0.1:6379> get test
          "aaaaaaa"

        ```
        
        + 业务处理结果最终输出
        
        ```shell script
          分布式锁已被重新获取，不执行删除
          lock.Lock 解锁失败，err= 分布式锁已被重新获取，不执行删除
        ```