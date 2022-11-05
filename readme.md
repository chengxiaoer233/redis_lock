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

* 加锁
```go
func Lock(ctx context.Context, key string, val string, timeout time.Duration)(ok bool, err error){

	ok ,err = client.SetNX(ctx, key, val, timeout).Result()
	if err != nil {
		fmt.Println("SetNX 错误，err=",err)
		return ok, err
	}

	return
}
```
* 解锁
```go
func Unlock(ctx context.Context,key string)(val int64, err error){

	val ,err = client.Del(ctx,key).Result()
	if err != nil {
		fmt.Println("解锁失败，err=",err)
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
    + 1：用户可以删除不是自己创建的分布式锁
    + 2：没有过期续期机制