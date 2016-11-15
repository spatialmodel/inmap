// Package requestcache provides functions for caching on-demand generated data.
package requestcache

import (
	"bytes"
	"context"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/golang/groupcache/lru"
)

// Cache is a holder for one or multiple caches.
type Cache struct {
	// requestChan receives incoming requests.
	requestChan chan *Request

	// requests holds the number of requests each individual cache has received.
	requests    []int
	requestLock sync.RWMutex
}

// NewCache creates a new set of caches for on-demand generated content, where
// processor is the function that creates the content, numProcessors is the
// number of processors that will be working in parallel, and cachefuncs are the
// caches to be used, listed in order of priority.
func NewCache(processor ProcessFunc, numProcessors int, cachefuncs ...CacheFunc) *Cache {
	c := &Cache{
		requestChan: make(chan *Request),
		requests:    make([]int, len(cachefuncs)+1),
	}

	in := c.requestChan
	out := in
	for i, cf := range cachefuncs {

		intermediate := make(chan *Request)
		go func(in chan *Request, i int) {
			// Track the number of requests received by this cache.
			for req := range in {
				c.requestLock.Lock()
				c.requests[i]++
				c.requestLock.Unlock()
				intermediate <- req
			}
		}(in, i)

		out = cf(intermediate)
		in = out
	}
	for i := 0; i < numProcessors; i++ {
		go func() {
			for req := range out {
				// Process the results
				c.requestLock.Lock()
				c.requests[len(cachefuncs)]++
				c.requestLock.Unlock()
				req.resultPayload, req.err = processor(req.ctx, req.requestPayload)
				req.returnChan <- req
			}
		}()
	}

	return c
}

// Requests returns the number of requests that each cache has received. The
// last index in the output is the number of requests received by the processor.
// So, for example, the miss rate for the first cache in c is r[len(r)-1] / r[0],
// where r is the result of this function.
func (c *Cache) Requests() []int {
	c.requestLock.Lock()
	out := make([]int, len(c.requests))
	copy(out, c.requests)
	defer c.requestLock.Unlock()
	return out
}

// ProcessFunc defines the format of functions that can be used to process
// a request payload and return a resultPayload.
type ProcessFunc func(ctx context.Context, requestPayload interface{}) (resultPayload interface{}, err error)

// Request holds information about a request that is to be handled either by
// a cache or a ProcessFunc.
type Request struct {
	ctx            context.Context
	requestPayload interface{}
	resultPayload  interface{}
	requestChan    chan *Request
	returnChan     chan *Request
	err            error
	funcs          []func(*Request)
	key            string
}

// NewRequest creates a new request where requestPayload is the input data
// that will be used to generate the results and key is a unique key.
func (c *Cache) NewRequest(ctx context.Context, requestPayload interface{}, key string) *Request {
	return &Request{
		requestPayload: requestPayload,
		returnChan:     make(chan *Request),
		requestChan:    c.requestChan,
		key:            key,
		ctx:            ctx,
	}
}

// Result sends the request for processing, waits for the result, and returns
// the result and any errors that occurred while
// processing.
func (r *Request) Result() (interface{}, error) {
	r.requestChan <- r
	rr := <-r.returnChan
	return rr.resultPayload, rr.finalize()
}

// finalize runs any clean-up functions that need to be run after the results
// have been generated and returns whether any errors have occurred.
func (r *Request) finalize() error {
	if r.err != nil {
		return r.err
	}
	for len(r.funcs) > 0 {
		f := r.funcs[0]
		r.funcs = r.funcs[1:len(r.funcs)]
		f(r)
		if r.err != nil {
			return r.err
		}
	}
	return r.err
}

// A CacheFunc can be used to store request results in a cache.
type CacheFunc func(in chan *Request) (out chan *Request)

// Deduplicate avoids duplicating requests.
func Deduplicate() CacheFunc {
	return func(in chan *Request) chan *Request {
		out := make(chan *Request)
		var dupLock sync.Mutex
		runningTasks := make(map[string][]*Request)

		dupFunc := func(req *Request) {
			dupLock.Lock()
			reqs := runningTasks[req.key]
			for i := 1; i < len(reqs); i++ {
				reqs[i].returnChan <- req
			}
			delete(runningTasks, req.key)
			dupLock.Unlock()
		}

		go func() {
			for req := range in {
				dupLock.Lock()
				if _, ok := runningTasks[req.key]; ok {
					// This task is currently running, so add it to the queue.
					runningTasks[req.key] = append(runningTasks[req.key], req)
					dupLock.Unlock()
				} else {
					// This task is not currently running, so add it to the beginning of the
					// queue and pass it on.
					runningTasks[req.key] = []*Request{req}
					req.funcs = append(req.funcs, dupFunc)
					dupLock.Unlock()
					out <- req
				}
			}
		}()
		return out
	}
}

// Memory manages an in-memory cache of results, where maxEntries is the
// max number of items in the cache. If the results returned by this cache
// are modified by the caller, they may also be modified in the cache.
func Memory(maxEntries int) CacheFunc {
	return func(in chan *Request) chan *Request {
		out := make(chan *Request)
		cache := lru.New(maxEntries)

		// cacheFunc adds the data to the cache and is sent along
		// with the request if the data is not in the cache
		cacheFunc := func(req *Request) {
			cache.Add(req.key, req.resultPayload)
		}

		go func() {
			for req := range in {
				if d, ok := cache.Get(req.key); ok {
					// If the item is in the cache, return it
					req.resultPayload = d
					req.returnChan <- req
				} else {
					// otherwise, add the request to the cache and send the request along.
					req.funcs = append(req.funcs, cacheFunc)
					out <- req
				}
			}
		}()
		return out
	}
}

// MarshalGob marshals an interface to a byte array and fulfills
// the requirements for the Disk cache marshalFunc input.
func MarshalGob(data interface{}) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	e := gob.NewEncoder(w)
	if err := e.Encode(data); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// UnmarshalGob unmarshals an interface from a byte array and fulfills
// the requirements for the Disk cache unmarshalFunc input.
func UnmarshalGob(b []byte) (interface{}, error) {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	var data interface{}
	if err := d.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// FileExtension is appended to request key names to make
// up the names of files being written to disk.
var FileExtension = ".dat"

// Disk manages an on-disk cache of results, where dir is the
// directory in which to store results, marshalFunc is the function to
// be used to marshal the data object to binary form, and unmarshalFunc
// is the function to be used to unmarshal the data from binary form.
func Disk(dir string, marshalFunc func(interface{}) ([]byte, error), unmarshalFunc func([]byte) (interface{}, error)) CacheFunc {
	return func(in chan *Request) chan *Request {

		out := make(chan *Request)

		// This function writes the data to the disk after it is
		// created, and is sent along with the request if the data is
		// not in the cache.
		writeFunc := func(req *Request) {
			fname := filepath.Join(dir, req.key+FileExtension)
			w, err := os.Create(fname)
			if err != nil {
				req.err = err
				return
			}
			defer w.Close()
			b, err := marshalFunc(&req.resultPayload)
			if err != nil {
				req.err = err
				return
			}
			if _, err = w.Write(b); err != nil {
				req.err = err
				return
			}
		}

		go func() {
			for req := range in {
				fname := filepath.Join(dir, req.key+FileExtension)

				f, err := os.Open(fname)
				if err != nil {
					// If we can't open the file, assume that it doesn't exist and Pass
					// the request on.
					req.funcs = append(req.funcs, writeFunc)
					out <- req
					continue
				}
				b, err := ioutil.ReadAll(f)
				if err != nil {
					// If we can't read the file, assume that there is some problem with
					// it and pass the request on.
					req.funcs = append(req.funcs, writeFunc)
					out <- req
					continue
				}
				data, err := unmarshalFunc(b)
				if err != nil {
					// There is some problem with the file. Pass the request on to
					// recreate it.
					req.funcs = append(req.funcs, writeFunc)
					out <- req
					continue
				}
				if err := f.Close(); err != nil {
					req.err = err
				}
				// Successfully retrieved the result. Now add it to the request
				// so it is stored in the cache and return it to the requester.
				req.resultPayload = data
				req.returnChan <- req
			}
		}()
		return out
	}
}
