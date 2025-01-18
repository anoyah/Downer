package http

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"github.com/go-resty/resty/v2"
)

type (
	Client struct {
		http  *resty.Client
		proxy bool
	}
)

func NewClient() (*Client, error) {
	client := resty.New()

	return &Client{
		http:  client,
		proxy: false,
	}, nil
}

func (c *Client) SetProxy(proxy string) error {
	_, err := url.ParseRequestURI(proxy)
	if err != nil {
		return err
	}

	c.http = c.http.SetProxy(proxy)
	return nil
}

func (c *Client) check() error {
	if !c.proxy {
		proxy := os.Getenv("https_proxy")
		return c.SetProxy(proxy)
	}

	return nil
}

type Response struct {
	body []byte
	size int64

	Header http.Header
}

func (r *Response) Body() []byte {
	return r.body
}

func (r *Response) Size() int64 {
	return r.size
}

// func (c *Client) SetHeader(key string, value string){

// }

func (c *Client) Do(ctx context.Context, url string, opts ...Option) (*Response, error) {
	response, err := c.do(ctx, url, opts...)
	if err != nil {
		return nil, err
	}

	return &Response{
		body:   response.Body(),
		size:   response.Size(),
		Header: response.Header(),
	}, nil
}

func (c *Client) do(ctx context.Context, url string, opts ...Option) (*resty.Response, error) {
	if err := c.check(); err != nil {
		return nil, err
	}

	var header Header
	for _, opt := range opts {
		opt(&header)
	}

	client := c.http.R()
	if header.accept != "" {
		client = client.SetHeader("accept", header.accept)
	}
	if header.authToken != "" {
		client = client.SetAuthToken(header.authToken)
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type Header struct {
	Url       string
	accept    string
	authToken string
}

type Option func(*Header)

func SetAccept(value string) Option {
	return func(h *Header) {
		h.accept = value
	}
}

func SetAuthToken(value string) Option {
	return func(h *Header) {
		h.authToken = value
	}
}

func (c *Client) Header(ctx context.Context, url string) (*Header, error) {
	response, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}

	header := response.Header()

	return &Header{
		Url: header.Get("url"),
	}, nil
}
