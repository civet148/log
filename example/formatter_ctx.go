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
	log.EnableShowStack(log.LevelError, log.LevelFatal)
}

type Request struct {
	Method string
	Body   string
}

func main() {
	ctx := log.NewContext(nil)
	defer log.PrintContext(ctx)
	reply, err := grpcMethodCall(ctx, &Request{
		Method: "GetUserList",
		Body:   `body data`,
	})
	if err != nil {
		return
	}
	log.WithFields(ctx, "reply", reply)
}

func grpcMethodCall(ctx context.Context, req interface{}) (res interface{}, err error) {
	log.WithPrintf(ctx, "gprc request: %+v", req)
	time.Sleep(1 * time.Second)
	log.WithFields(ctx, "trace_id", 100000000000086, "key2", 2, "key3", "3")
	if err = occurError(ctx); err != nil {
		return nil, log.WithError(ctx, err)
	}
	return "success", nil
}

func occurError(ctx context.Context) (err error) {
	return fmt.Errorf("network connection failed")
}
