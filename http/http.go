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

	Config struct {
		proxy string
	}
	ClientOption func(*Config)
)

// WithProxy set proxy to send http request
func WithProxy(proxy string) ClientOption {
	return func(c *Config) {
		c.proxy = proxy
	}
}

// NewClient create request struct with resty third
func NewClient(opts ...ClientOption) (*Client, error) {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	request := resty.New()
	client := &Client{
		http: request,
	}

	if cfg.proxy != "" {
		client.SetProxy(cfg.proxy)
	}

	return client, nil
}

// SetProxy set proxy with client
func (c *Client) SetProxy(proxy string) error {
	_, err := url.ParseRequestURI(proxy)
	if err != nil {
		return err
	}

	c.http = c.http.SetProxy(proxy)
	c.proxy = true

	return nil
}

func (c *Client) Do(ctx context.Context, url string, opts ...HeaderOption) (*Response, error) {
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

func (c *Client) do(_ context.Context, url string, opts ...HeaderOption) (*resty.Response, error) {
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

type Header struct {
	Url       string
	accept    string
	authToken string
}

type HeaderOption func(*Header)

func SetAccept(value string) HeaderOption {
	return func(h *Header) {
		h.accept = value
	}
}

func SetAuthToken(value string) HeaderOption {
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
