package main

import (
	"context"
	"fmt"
	"time"

	"github.com/civet148/log/v2"
)

func init() {
	log.SetLevel(log.LevelDebug)
	log.SetJsonFormatter()
	log.SetFileName("output-json.log") //同时输出到文件
	log.EnableShowStack()
}

type Request struct {
	Method string
	Body   string
}

func main() {
	ctx := log.NewContext(nil)
	grpcMethodCall(ctx, &Request{
		Method: "GetUserList",
		Body:   `{"id":100932}`,
	})
	log.PrintContext(ctx)
}

func grpcMethodCall(ctx context.Context, req interface{}) (res interface{}, err error) {
	log.WithPrintf(ctx, "my name is %s", "lory")
	time.Sleep(3 * time.Second)
	log.WithFields(ctx, "key1", 1, "key2", 2, "key3", "3")
	if err = occurError(ctx); err != nil {
		return nil, log.WithError(ctx, err)
	}
	return struct{}{}, nil
}

func occurError(ctx context.Context) (err error) {
	return fmt.Errorf("network connection failed")
}
