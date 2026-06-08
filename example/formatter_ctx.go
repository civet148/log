package main

import "github.com/civet148/log/v2"

func main() {
	ctx := log.NewContext(nil)
	log.WithPrintf(ctx, "hello")
	log.WithPrintFields(ctx, "key1", 1, "key2", 2, "key3")
	log.WithPrintf(ctx, "world")
	log.PrintContext(ctx)
}
