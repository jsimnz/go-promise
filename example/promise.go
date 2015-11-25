package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/jsimnz/promise"
)

func someErrorFunc() interface{} {
	fmt.Println("Running workload...")
	time.Sleep(time.Second * 2)
	fmt.Println("Finished workload!")
	return errors.New("Test error")
}

func someValueFunc() interface{} {
	fmt.Println("Running workload...")
	time.Sleep(time.Second * 3)
	fmt.Println("Finished workload!")
	return "Some result"
}

func main() {
	p1 := promise.NewPromise(func(resolve promise.ResolveFn, reject promise.RejectFn) {
		//var err error
		err := errors.New("Some error")
		fmt.Println("Running process...")
		time.Sleep(time.Second * 2)
		fmt.Println("Finished work!")
		if err != nil {
			reject(err)
		} else {
			resolve("My Result")
		}
	})

	p1.Then(func(result ...interface{}) {
		fmt.Println(result)
	}, func(err ...interface{}) {
		fmt.Println(err)
	})

	fmt.Printf("\n==================\n\n")

	p2 := promise.NewPromiseFunc(someValueFunc)
	p2.Then(func(result ...interface{}) {
		fmt.Println(result)
	}, func(errs ...interface{}) {
		fmt.Println(errs)
	})

	fmt.Printf("\n==================\n\n")

	p3 := promise.NewPromiseFunc(someValueFunc)
	p4 := promise.NewPromiseFunc(someErrorFunc)

	p5 := promise.All(p3, p4)
	p5.Then(func(result ...interface{}) {
		fmt.Println("MULTI: Success!")
		fmt.Println(result)
	}, func(errs ...interface{}) {
		fmt.Println("MULTI: Error!")
		fmt.Println(errs)
	})
}
