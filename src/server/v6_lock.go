package lock

import (
	"context"
	"errors"
	"fmt"
	"redis_lock/utils"
	"time"
)

// Lock 是尽可能重试减少加锁失败的可能
// Lock 会在超时或者锁正被人持有的时候进行重试
// 最后返回的 error 使用 errors.Is 判断，可能是：
// - context.DeadlineExceeded: Lock 整体调用超时
// - ErrFailedToPreemptLock: 超过重试次数，但是整个重试过程都没有出现错误
// - DeadlineExceeded 和 ErrFailedToPreemptLock: 超过重试次数，但是最后一次重试超时了
// 你在使用的过程中，应该注意：
// - 如果 errors.Is(err, context.DeadlineExceeded) 那么最终有没有加锁成功，谁也不知道
// - 如果 errors.Is(err, ErrFailedToPreemptLock) 说明肯定没成功，而且超过了重试次数
// - 否则，和 Redis 通信出了问题

// 加锁的时候，也会存在失败，需要失败重试
func (c *Client) V6Lock(ctx context.Context, key string, val string, expiration time.Duration, timeout time.Duration, luaLock string, retry utils.RetryStrategy) (lock *Lock, ok bool, err error) {

	var timer *time.Timer
	defer func() {
		if timer != nil { // 程序结束时需要关闭定时器
			timer.Stop()
		}
	}()

	for {

		lCtx, cancel := context.WithTimeout(ctx, timeout)
		res, err := c.client.Eval(lCtx, luaLock, []string{key}, val, expiration.Seconds()).Result()
		cancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {

			// 非超时错误，那么基本上代表遇到了一些不可挽回的场景，所以没太大必要继续尝试了
			// 比如说 Redis server 崩了，或者 EOF 了
			return nil, false, err
		}

		// 加锁成功
		if res == "OK" {
			return &Lock{client: c.client, key: key, val: val, expiration: expiration}, true,nil
		}

		//  重试
		interval, ok := retry.Next()
		if !ok {
			if err != nil {
				err = fmt.Errorf("最后一次重试错误: %w", err)
			} else {
				err = fmt.Errorf("锁被人持有: %s", "加锁失败")
			}
			return nil, false, fmt.Errorf("rlock: 重试机会耗尽，%w", err)
		}

		if timer == nil { // 第一次重试
			timer = time.NewTimer(interval)
		} else { // 后面重试，直接重置timer
			timer.Reset(interval)
		}

		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, true, ctx.Err()
		}
	}
}
