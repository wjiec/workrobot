package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	uuid "github.com/satori/go.uuid"

	"github.com/pkg/errors"
)

// Uploader represents a wxUploadReceipt uploader
type Uploader struct {
	hc       *http.Client
	endpoint string
}

// UploadFromReader upload a wxUploadReceipt from reader
func (u *Uploader) UploadFromReader(reader io.Reader) (*Media, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filename := uuid.NewV4().String()
	if f, ok := reader.(interface{ Name() string }); ok {
		filename = f.Name()
	}

	part, err := writer.CreateFormFile("media", filepath.Base(filename))
	if err != nil {
		return nil, errors.Wrap(err, "cannot create multipart")
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, errors.Wrap(err, "reader unreadable")
	}

	if err = writer.Close(); err != nil {
		return nil, errors.Wrap(err, "multipart not writable")
	}

	req, err := http.NewRequest(http.MethodPost, u.endpoint, &body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return u.upload(req)
}

// UploadFromFile upload a wxUploadReceipt from filename
func (u *Uploader) UploadFromFile(filename string) (*Media, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open file")
	}
	defer func() { _ = f.Close() }()

	return u.UploadFromReader(f)
}

// wxUploadReceipt represents an upload response
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#文件上传接口
type wxUploadReceipt struct {
	Id        string `json:"media_id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`

	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
}

// upload execute an request and check http response
func (u *Uploader) upload(req *http.Request) (*Media, error) {
	resp, err := u.hc.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unreadable http response")
	}

	var receipt wxUploadReceipt
	if err := json.Unmarshal(body, &receipt); err != nil {
		return nil, errors.Wrap(err, "invalid http response")
	}

	if receipt.Code != 0 {
		return nil, errors.New(fmt.Sprintf("%d: %s", receipt.Code, receipt.Message))
	}

	ts, err := strconv.ParseInt(receipt.CreatedAt, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid create_at response")
	}

	return &Media{Id: receipt.Id, Type: receipt.Type, CreatedAt: ts}, nil
}

// Media represents an uploaded wxUploadReceipt
type Media struct {
	Id        string `json:"media_id"`
	Type      string `json:"type"`
	CreatedAt int64  `json:"created_at"`
}

// New create a wxUploadReceipt uploader
func New(hc *http.Client, endpoint string) *Uploader {
	return &Uploader{hc: hc, endpoint: endpoint}
}
