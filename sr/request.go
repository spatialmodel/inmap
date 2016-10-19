package sr

import "golang.org/x/net/context"

// Request holds information about a request that is to be handled by a
// cluster
type Request struct {
	ctx            context.Context
	requestPayload *IOData
	resultPayload  IOData
	service        string
	requestChan    chan *Request
	returnChan     chan *Request
	err            error
}

// NewRequest creates a new request where service is the RPC service that
// should be called and requestPayload is the input data
// that will be used to generate the results.
// The result of the RPC call must be of the same type as the request payload.
func (c *Cluster) NewRequest(ctx context.Context, service string, requestPayload *IOData) *Request {
	return &Request{
		requestPayload: requestPayload,
		returnChan:     make(chan *Request, 1),
		requestChan:    c.requestChan,
		service:        service,
		ctx:            ctx,
	}
}

// Send sends the request for processing,
func (r *Request) Send() {
	r.requestChan <- r
}

// Result waits for the result, and returns
// the result and any errors that occurred while
// processing. Result should be called after send.
func (r *Request) Result() (*IOData, error) {
	rr := <-r.returnChan
	return &rr.resultPayload, rr.err
}
