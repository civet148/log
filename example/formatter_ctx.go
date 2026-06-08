package main

import "github.com/civet148/log/v2"

func init() {
	log.SetLevel(log.LevelDebug)
	log.SetJsonFormatter()
}

func main() {
	ctx := log.NewContext(nil)
	log.WithPrintf(ctx, "hello")
	log.WithFields(ctx, "key1", 1, "key2", 2, "key3")
	log.WithPrintf(ctx, "world")
	log.WithPrintf(ctx, "my name is %s", "lory")
	log.PrintContext(ctx)
}
