package tool

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// 使用外部帶入ctx andd cancel
func WaitShutdown(callback func()) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal)

	go func() {

		signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt, os.Kill)

		defer signal.Stop(c)

		sigReceived := <-c
		fmt.Println("[notice] received signal:", sigReceived)
		cancel()
		callback()
	}()

	return ctx
}
