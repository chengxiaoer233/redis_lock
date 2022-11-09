-- 设置成功返回 1
-- 当key不存在或者不能为key设置过期时间时(比如在低于 2.1.3 版本的Redis中你尝试更新key的过期时间)返回 0 。

if redis.call("get", KEYS[1]) == ARGV[1]
then
    return redis.call("expire", KEYS[1], ARGV[2])
else
    return 0
end