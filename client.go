package workrobot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	uploader "github.com/wjiec/workrobot/media"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

const (
	// default robot webhook gateway
	DefaultSendGateway = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"
	// default upload gateway
	DefaultUploadGateway = "https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media"
)

// Client represents a robot client
type Client struct {
	hc *http.Client

	key     string
	webhook string
}

// Send send messages to the group in order, when error occurs
// will be returns immediately and skip the rest of the messages
func (c *Client) Send(messages ...Messager) error {
	for _, msg := range messages {
		if err := c.doSend(context.Background(), msg); err != nil {
			return err
		}
	}

	return nil
}

// SendConcurrency send message to the group concurrency, and error will returns
// according to fastFail, returns immediately when fastFail is true, aggregation
// errors otherwise
//
// concurrency limit: 20/min, 2/sec
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#消息发送频率限制
func (c *Client) SendConcurrency(fastFail bool, messages ...Messager) (err error) {
	var wg sync.WaitGroup
	var failed sync.Once
	errs := make(chan error, len(messages))
	ctx, cancel := context.WithCancel(context.Background())

	for _, msg := range messages {
		wg.Add(1)
		go func(msg Messager) {
			defer wg.Done()
			if se := c.doSend(ctx, msg); se != nil {
				if fastFail {
					failed.Do(func() {
						cancel()
						err = se
					})
				} else if se != context.Canceled {
					errs <- se
				}
			}
		}(msg)
	}

	wg.Wait()
	for {
		select {
		case e := <-errs:
			err = multierr.Append(err, e)
		default:
			close(errs)
			return
		}
	}
}

// wxSendReceipt represents an receipt from workWx
type wxSendReceipt struct {
	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
}

// Error build error message and returns when error occurs
func (r *wxSendReceipt) Error() string {
	return fmt.Sprintf("%d: %s", r.Code, r.Message)
}

// doSend send single message to the group
func (c *Client) doSend(ctx context.Context, msg Messager) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhook, bytes.NewReader(msg.Message()))
	if err != nil {
		return errors.Wrap(err, "bad request")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return errors.Wrap(err, "http request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "unreadable http response")
	}

	var receipt wxSendReceipt
	if err := json.Unmarshal(bs, &receipt); err != nil {
		return errors.Wrap(err, "wrong http response data")
	}

	if receipt.Code != 0 {
		return &receipt
	}
	return nil
}

// Uploader returns the uploader for current robot
func (c *Client) Uploader() *uploader.Uploader {
	endpoint, _ := url.Parse(DefaultUploadGateway)

	q := endpoint.Query()
	q.Add("key", c.key)
	q.Add("type", "file")
	endpoint.RawQuery = q.Encode()

	return uploader.New(c.hc, endpoint.String())
}

// ClientOption represents additional robot configuration
type ClientOption func(*Client) error

// WithHttpClient override http client for robot request
func WithHttpClient(hc *http.Client) ClientOption {
	return func(client *Client) error {
		client.hc = hc
		return nil
	}
}

// WithWebhook override robot webhook address
func WithWebhook(webhook string) ClientOption {
	return func(client *Client) error {
		api, err := url.Parse(webhook)
		if err != nil {
			return err
		}

		client.webhook = api.String()
		return nil
	}
}

// NewClient create a instance of robot
func NewClient(key string, options ...ClientOption) (*Client, error) {
	c := &Client{hc: http.DefaultClient, webhook: Webhook(key), key: key}
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Webhook build webhook address from robot key
func Webhook(key string) string {
	api, _ := url.Parse(DefaultSendGateway)

	q := api.Query()
	q.Add("key", key)

	api.RawQuery = q.Encode()
	return api.String()
}
