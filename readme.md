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
    
    + 加锁后，通过在终端手动删除key（模拟锁被别人删除了）
        
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
    
    + 加锁后，通过在终端手动修改key对应的val值（模拟锁被别人重新获取）
                
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
* 存在的问题 
    + （1）删除锁的时候，先判断锁是否存在，再删除，非原子操作，还是会删除别人的锁     
   
#### 版本三：释放分布式锁的时候，使用lua脚本,实现原子操作
* lua + redis  可以解决redis中的原子操作问题  

* 释放分布式锁

    + 直接调用lua脚本，先判断锁是否是自己的，如果是则删除，否则退出
    ```go
    // 释放分布式锁
    // 需要用加锁返回的实体才可以调用
    func (l *Lock) V3Unlock(ctx context.Context, luaUnlock string)(int64, error){
    
    	// 执行lua脚本，原子操作，根据返回的值做不同的处理逻辑
    	val, err := l.client.Eval(ctx,luaUnlock,[]string{l.key},l.val).Int64()
    	if err != nil {
    		fmt.Println("l.client.Eval err=",err)
    		return val,err
    	}
    
    	// val返回0时表示key不存在或者key对应的val值不对
    	return val,err
    }
    ```  
    
    + lua脚本释放分布式锁
    ```lua
   -- redis.call() 从lua脚本中调用redis方法
   
   -- KEYS[1]，表示输如的第一个key
   -- ARGV[1]，为输入的第一个参数
   
   if redis.call("get", KEYS[1]) == ARGV[1]
   then
       -- 相等则执行删除动作，并返回执行删除后的结果
       return redis.call("del", KEYS[1])
   else
       -- 返回0表示key不存在，或者key对应的val值已经被修改了
       return 0
   end
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
	_ "embed"
)

var (
	// 解释：通过go:embed命令，将unlock.lua中的内容格式化赋予到luaUnlock（string类型）
	// 注意：（1）go embed 只能嵌入当前目录或者子目录，不能嵌入上一级目录
	//      （2）go embed 不能再函数内部定义，需要再func 外面定义

	//go:embed lua/unlock.lua
	luaUnlock string
)

func main() {

	ctx := context.Background()
	for i := 0; i < 5; i++ {

		go func() {
			V3LockTest(ctx)
		}()
	}

	time.Sleep(1 * time.Hour)
}

// v3 采用lua脚本来实现分布式锁的解锁
func V3LockTest(ctx context.Context) {

	// 生成一个redis的client
	c := new(dao.Redis)
	redisClient := c.NewClient()

	// 生成一个 Cmdable client，这里也是可以传入redis.ClusterClient的
	cClient := lock.NewClient(redisClient)

	key := "test3"
	// vale这里不能再随意设置，需要为uuid,后面删除的时候，需要对比此值是否一致，是则可以删除，否则不行
	val := uuid.New().String()
	timeout := time.Second * 60

	// 加锁,会返回一个lock实体和加锁函数对应的日志
	lock,ok, err := cClient.V3Lock(ctx, key, val, timeout)
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
	res, err := lock.V3Unlock(ctx,luaUnlock)
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

#### 版本四：在v3的基础上增加手动续约机制，用户可以手动针对key续约
* refresh函数
```go
    // 手动续约，执行refresh lua脚本
    // 本质就是针对某个key重新设置过期时间

    func (l *Lock) refresh(ctx context.Context, luaRefresh string) error {
    
    	res, err := l.client.Eval(ctx,luaRefresh,[]string{l.key},l.val,l.expiration.Seconds()).Int64()
    	if err != nil {
    		return err
    	}
    
    	if res != 1 {
    		fmt.Println("获取分布式锁失败")
    	}
    
    	return nil
    }
```

* refresh.lua函数
```lua
    -- 先检测key对应的val是否和设置的一样
    -- 不一样退出，一样则调用expire函数

    if redis.call("get", KEYS[1]) == ARGV[1]
    then
        return redis.call("expire", KEYS[1], ARGV[2])
    else
        return 0 
    end
```

* 在手动续约的基础上，增加最大续约次数和自动以时间间隔
```go
// 手动续约,最大续约次数，和续约间隔
func (l *Lock) timeToRefresh(tryLockCount int, interval time.Duration, luaRefresh string) {

	// 初始化一个chan，用户接受业务退出信号
	end := make(chan struct{},1)

	// 启动一个协程去执行续约任务
	go func (){

		tmpCount := 0
		ticker := time.NewTicker(time.Second * interval)
		for {
			select {
			case <-ticker.C: // 定时时间到，需要续约
				{
					tmpCount++ // 续约次数加一，超过最大续约次数就退出
					if tmpCount > tryLockCount {
						return
					}

					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					err := l.refresh(ctx, luaRefresh)
					cancel()

					// 错误处理
					if err == context.DeadlineExceeded {
						// 超时，按照道理来说，你应该立刻重试
						// 超时之下可能续约成功了，也可能没成功
					}

					if err != nil && err != context.DeadlineExceeded {
						// 其它错误，你要考虑这个错误能不能继续处理
						// 如果不能处理，你怎么通知后续业务中断？
					}
				}
			case <- end : // 业务主动退出了
				{
					fmt.Printf("业务主动退出了")
					return
				}
			}
		}
	}()

	// 这里模拟业务逻辑
	time.Sleep(30 * time.Second)

	// 业务结束
	end <- struct{}{} // 发送结束信号
}
```

#### 版本五：在v4手动续约的基础上，增加自动续约机制

```go

```


#### 常见问题
* （1）加锁的时候为什么需要设置超时时间？  
 
   主要是为了防止某个用户拿到锁后，由于业务逻辑问题或者实例崩溃，导致长期没释放锁，造成别的业务
   也没法使用此分布式锁
   
* （2）加锁的时候，为什么要设置一个uuid做为key的val，随意值行不行？
   
    不行，用户释放分布式锁的时候，需要判断此锁是不是自己的，用户不能删除别人的锁，用户
    每次删除前需要去获取该key对应的val值，由于val使用的是uuid，故只要此val和自己设定的时候是一致的
    则说明锁是自己的，自己可以释放，否则说名，锁已经被别人获取了，不能释放。本质我们需要一个唯一的值来确认
    当前锁是属于某个key的，由于uuid构造方便且简单，所以用了这个

* （3）加锁的时候为什么返回一个lock对象？
    分布式锁的原则是自己加锁的自己才能释放，别人不能释放，所以unlock方法不能是一个全局的，
    而是需要通过lock加锁成功后返回一个lock实体，用此实体才可以解锁。
    
* （4）释放锁的时候，怎么保证原子操作？
    先判断此锁是否是自己的，如果是则删除，不是原子操作删除的时候可能已经被释放了，
    要保障原子操作，只能把这两个逻辑用lua脚本实现，保证原子操作。     

* （5）过期时间设置多久设置比较合理？
    超时时间很难设定
    设置过长：崩溃了，其他实例长时间拿不到锁
    设置过短：业务没处理完，锁已经释放了
    不论设置多长时间，极端情况下，都是存在业务执行时间超过设定时间
    
* （6）为什么需要续约？
    锁还没有过期的时候，设置一次过期时间  
    + 过期时间不必设置的过程，自动续约会在要过期时帮气门设置好
    + 加入实例崩溃了，则改实例不会自动续约，过一段时间就会过期
    其他实例就可以拿到锁了

* （7）手动续约的问题？  
    + （1）多久续约一次 ？
        手动续约不必把过期时间设置过长，因为有续约机制，同时为了避免实例
        崩溃其他实例拿不到锁问题
        
    + （2）续约如果出现了问题，改怎么办 ？  
            + 超时error ？
                + 超时之下可能续约成功了，也可能没成功    
                + 重试几次，同时做好监控上报
                
            + 其他服务器error怎么办 ？
                + 要看业务场景，能不能继续处理，不能则发信号程序退出，同时回滚业务
            
    + （3）如果确认续约失败了，业务要怎么处理 ？
        无解，除非手动检测分布式锁，不然没有办法    
             
*  （8）自动续约 ,尽量不让业务使用自动续约机制，不好把控
    + 隔多久续约，续多长 ？  
        让用户根据业务决定隔多久续约一次，每次续约多长直接复用原始的设定值
        
    + 超时如何处理？以及设置多久的超时时间 ？  
        超时一般是偶发的，出现概率很小，此时我们可以选择重试。缺点就是如果真的是
        网络崩溃或网络不通，则会导致不断的重试。超时时间的设定让用户指定。
        
    + 如果自动续约失败了，怎么办  ？
        只处理超时引起的续约失败，其他失败让自己自己决定如何处理
        
    + 自动续约需要设置上限吗？ 
        接口参数暴露出来，让业务自己定，如果业务有此需求，让业务自己决定  
        
*   （9）加锁重试，也需要引入lua脚本
    + 加锁的时候，也会存在失败，所以需要重试
    + 重试次数？重试间隔？什么情况下需要重试？什么时候不用重试？
    
    + 重试时需要检测当前的key对应的value是否存在，
        + 存在且相等则上次加锁成功了，然后其他超时引起整体超时  
        + 存在但不相等则锁被人占领了
        + 不存在则直接加速  