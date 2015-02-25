package scope_test

import (
	"fmt"
	"time"

	"euphoria.io/scope"
)

func ExampleBreakpointer() {
	root := scope.New()

	// A function that returns an error, which we want to simulate.
	performSomeTask := func(arg string) (string, error) { return arg, nil }

	// A function that we want to test the error handling of.
	errorHandler := func(ctx scope.Context, arg string) (string, error) {
		if err := ctx.Check(performSomeTask, arg); err != nil {
			return "", err
		}
		return performSomeTask(arg)
	}

	// Set a breakpoint on a particular invocation of performSomeTask.
	ctrl := root.Breakpoint(performSomeTask, "fail")

	// Other invocations should proceed as normal.
	_, err := errorHandler(root, "normal")
	if err != nil {
		fmt.Println("unexpected error:", err)
	}

	// Our breakpoint should allow us to inject an error. To control it
	// we must spin off a goroutine.
	go func() {
		<-ctrl // synchronize at beginning of errorHandler
		ctrl <- fmt.Errorf("test error")
	}()

	if _, err := errorHandler(root, "fail"); err == nil || err.Error() != "test error" {
		fmt.Println("unexpected result:", err)
	}

	// We can also inject an error by terminating the context.
	go func() {
		<-ctrl
		root.Cancel()
	}()

	if _, err := errorHandler(root, "fail"); err != scope.Cancelled {
		fmt.Println("unexpected result:", err)
	}
}

func ExampleContext_cancellation() {
	ctx := scope.New()
	go func() {
		time.Sleep(5 * time.Second)
		ctx.Cancel()
	}()

	for {
		t := time.After(time.Second)
		select {
		case <-ctx.Done():
			break
		case <-t:
			fmt.Println("tick")
		}
	}
	fmt.Println("finished with", ctx.Err())
}
