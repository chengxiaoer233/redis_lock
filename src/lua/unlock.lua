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
