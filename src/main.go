package main

import (
	"context"
	lock "redis_lock/server"
	"time"
)

func main(){

	ctx := context.Background()
	for i := 0 ; i < 10 ; i++ {

		go func(){
			lock.Lock(ctx,"test","val",time.Second * 10)
		}()
	}

	time.Sleep(time.Second * 10)
}
