package promise

import "reflect"

const DONE_CHAN_BUFFER_SIZE = 100

// Promise interface
// Anything that can be called with Then results in a promise
// This allows the implementation of promises to be abstracted
// away. This can be seen in the difference between the
// singlePromise struct vs multiPromise struct.
type Promise interface {
	Then(func(...interface{}), func(...interface{}))
}

// Function type to be called when we 'resolve' a promise
type ResolveFn func(...interface{})

// Function type to be called when we 'reject' a promise
type RejectFn func(...interface{})

type packedResults struct {
	results []interface{}
}

// A promise function is always accepts a resolve and reject function
// as its parameters. However as can be seen with NewPromiseFunc, we
// can automatically create this type of function by wrapping
// any normal function.
type PromiseFn func(ResolveFn, RejectFn)

type singlePromise struct {
	succ  chan packedResults // Success chanel
	err   chan packedResults // Error channel
	d     chan bool          // Done channel
	fnRaw interface{}        // Raw promise function
	eval  PromiseFn          // Properly formatted/wrapped promise function

}

// Create a new Promise using the PromiseFn type
func NewPromise(fn PromiseFn) Promise {
	p := &singlePromise{
		succ: make(chan packedResults),
		err:  make(chan packedResults),
		d:    make(chan bool, DONE_CHAN_BUFFER_SIZE),
		eval: fn,
	}

	// New resolveFn scoped to our new instance of Promise
	var resolveFn ResolveFn = func(results ...interface{}) {
		packed := packedResults{
			results: results,
		}
		p.d <- true
		p.succ <- packed
	}

	// New rejectFn scoped to our new instance of Promise
	var rejectFn RejectFn = func(results ...interface{}) {
		packed := packedResults{
			results: results,
		}
		p.err <- packed
	}

	go p.eval(resolveFn, rejectFn)

	return p
}

// Create a new promise from any function, and automatically
// wrap it with the resolve/reject PromiseFn type
func NewPromiseFunc(fn interface{}) Promise {
	fnval := reflect.ValueOf(fn)
	if fnval.Kind() != reflect.Func {
		return nil
	}
	var wrappedFn PromiseFn = func(resolve ResolveFn, reject RejectFn) {
		outputsRaw := fnval.Call([]reflect.Value{})
		var outputs []interface{}

		for _, output := range outputsRaw {
			outputs = append(outputs, output.Interface())
		}

		if len(outputs) == 0 {
			resolve() // resolve promise and exit
			return
		} else if len(outputs) == 1 {
			if err, ok := outputs[0].(error); ok && err != nil {
				// returned an error
				reject(err)
				return
			} else {
				resolve(outputs...)
			}
		} else {
			resolve(outputs...)
		}
	}

	return NewPromise(wrappedFn)
}

// Handles the resolved/rejected promise
// TODO: Promise chaining via Then
func (p *singlePromise) Then(succFn func(...interface{}), errFn func(...interface{})) {

	//var pnext *Promise

	select {
	case result := <-p.succ:
		// success
		succFn(result.results...)
	case result := <-p.err:
		// error
		errFn(result.results...)
	}
}

type multiPromise struct {
	promises []Promise
}

// Create a promise that is a collection of multiple promises
// that resolves when ALL promises have resolved, or rejects
// when the first of any of the promises rejects.
func All(promises ...Promise) Promise {
	mp := &multiPromise{
		promises: promises,
	}
	return mp
}

func (mp *multiPromise) Then(succFn func(...interface{}), errFn func(...interface{})) {
	results := make([][]interface{}, len(mp.promises))
	errs := make([][]interface{}, len(mp.promises))
	successDone := make(chan bool)
	errDone := make(chan int)

	successCollector := func(index int) func(...interface{}) {
		return func(res ...interface{}) {
			results[index] = res
			successDone <- true
		}
	}
	errorCollector := func(index int) func(...interface{}) {
		return func(e ...interface{}) {
			errs[index] = e
			errDone <- index
		}
	}
	for i, p := range mp.promises {
		go p.Then(successCollector(i), errorCollector(i))
	}

	successCounter := 0

monitor:
	for {
		select {
		case <-successDone:
			successCounter++
			if successCounter == len(mp.promises) {
				break monitor
			}
		case index := <-errDone:
			errFn(errs[index])
			return
		default:
			//pass
		}
	}

	var flatResults []interface{}
	for _, res := range results {
		flatResults = append(flatResults, res...)
	}

	succFn(flatResults)
	return
}
