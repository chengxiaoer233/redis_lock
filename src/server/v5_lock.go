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
func (c *Client) V5Lock(ctx context.Context, key string, val string, timeout time.Duration) (lock *Lock, ok bool, err error) {

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
func (l *Lock) V5Unlock(ctx context.Context, luaUnlock string) (int64, error) {

	defer func() {
		// 避免重复解锁引起 panic
		l.signalUnlockOnce.Do(func() {
			l.unlock <- struct{}{}
			close(l.unlock)
		})
	}()

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
func (l *Lock) v5refresh(ctx context.Context, luaRefresh string) error {

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

// 手动续约,增加最大续约次数，和续约间隔
func (l *Lock) AutoRefresh(tryLockCount int, interval time.Duration, timeout time.Duration, luaRefresh string) error{

	timer := time.NewTicker(interval)
	tmpCount := 0

	// 刷新超时 channel
	ch := make(chan struct{}, 1)
	defer func() {
		timer.Stop() // 关闭定时器
		close(ch)    // 关闭channel
	}()

	for {
		select {
			case <- timer.C:{ // 续约时间到

				tmpCount++ // 续约次数加一，超过最大续约次数就退出
				if tmpCount > tryLockCount {
					return errors.New("续约次数过多")
				}

				ctx ,cancel := context.WithTimeout(context.Background(),timeout)
				err := l.v5refresh(ctx, luaRefresh)
				cancel()

				// 错误处理
				if err != context.DeadlineExceeded {

					// 因为有两个可能的地方要写入数据，而 ch
					// 容量只有一个，所以如果写不进去就说明前一次调用超时了，并且还没被处理，
					// 与此同时计时器也触发了
					select {
						case ch <- struct{}{}:
						default:
					}
					continue
				}

				// 其他错误
				if err != nil {
					return err
				}
			}

			case <- ch: { // 加锁超时，重试逻辑

				tmpCount++ // 续约次数加一，超过最大续约次数就退出
				if tmpCount > tryLockCount {
					return errors.New("续约次数过多")
				}

				ctx ,cancel := context.WithTimeout(context.Background(),timeout)
				err := l.v5refresh(ctx, luaRefresh)
				cancel()

				// 错误处理
				if err != context.DeadlineExceeded {

					select {
					case ch <- struct{}{}:
					default:
					}
					continue
				}

				// 其他错误
				if err != nil {
					return err
				}
			}
		case <- l.unlock:{ // 业务主动释放了锁
			return nil
			}
		}
	}
}
