package workrobot

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	md "github.com/wjiec/workrobot/markdown"
)

var (
	// ErrMessageTooLong represents the message length exceeds the limit
	ErrMessageTooLong = errors.New("message too long")
	// ErrImageTooLarge represents the image size exceeds the limit(2m)
	ErrImageTooLarge = errors.New("image too large")
	// ErrTooManyArticle represents articles count more than 8
	ErrTooManyArticle = errors.New("too many articles")
)

const (
	TextMessageMaxLength     = 2048
	MarkdownMessageMaxLength = 4096
	MaxImageFileSize         = 2 * 1024 * 1024 // 2M
	MaxArticleCount          = 8
)

// Messager represents a message will sent
type Messager interface {
	Message() []byte
}

// Mention represents a mention message
type Mention struct {
	members []string
	mobiles []string

	all bool
}

// MentionAll make mentioned all group members
func (msg *Mention) MentionAll(all bool) *Mention {
	msg.all = all

	return msg
}

// MentionMember make mentioned someone in group
func (msg *Mention) MentionMember(member string) *Mention {
	msg.members = append(msg.members, member)

	return msg
}

// MentionMobile make mentioned someone by mobile in group
func (msg *Mention) MentionMobile(mobile string) *Mention {
	msg.mobiles = append(msg.mobiles, mobile)

	return msg
}

// Message implement Messager and build message only mention
func (msg *Mention) Message() []byte {
	return msg.payload().Build()
}

// payload build request payload to generate message
func (msg *Mention) payload() *payload {
	data := &payload{MessageType: "text", Text: &text{MentionMembers: msg.members, MentionMobiles: msg.mobiles}}
	if msg.all && len(data.Text.MentionMobiles) != 0 && len(data.Text.MentionMembers) == 0 {
		data.Text.MentionMobiles = append(data.Text.MentionMobiles, "@all")
	} else if msg.all {
		data.Text.MentionMembers = append(data.Text.MentionMembers, "@all")
	}

	return data
}

// NewMention create a mention message without error
func NewMention(members []string, mobiles []string, all bool) *Mention {
	var mention Mention
	for _, member := range members {
		mention.MentionMember(member)
	}
	for _, mobile := range mobiles {
		mention.MentionMobile(mobile)
	}
	mention.MentionAll(all)

	return &mention
}

// Text represents a text and mention message
type Text struct {
	Mention

	content string
}

// Content sets the message content, only pure text, and max
func (msg *Text) Content(content string) error {
	if len(content) > TextMessageMaxLength {
		return ErrMessageTooLong
	}

	msg.content = content
	return nil
}

// Message implement Messager and build message with content and mention
func (msg *Text) Message() []byte {
	data := msg.payload()
	data.Text.Content = msg.content

	return data.Build()
}

// NewText create a text message
func NewText(content string) (*Text, error) {
	var txt Text
	return &txt, txt.Content(content)
}

// Markdown represents a pure markdown message
type Markdown struct {
	len   int
	lines []string
}

// RawContent sets the raw markdown text
func (msg *Markdown) RawContent(raw string) error {
	if len(raw) > MarkdownMessageMaxLength {
		return ErrMessageTooLong
	}

	msg.len = len(raw)
	msg.lines = []string{raw}
	return nil
}

// AddSegmentLine add a segment as line
func (msg *Markdown) AddSegmentLine(s md.Segment) error {
	return msg.AddLine(s.String())
}

// AddLine add a text as line
func (msg *Markdown) AddLine(l string) error {
	if msg.len+len(l)+1 > MarkdownMessageMaxLength {
		return ErrMessageTooLong
	}

	msg.len += len(l) + 1 // \n
	msg.lines = append(msg.lines, l)
	return nil
}

// Message implement Messager and build message with markdown content
func (msg *Markdown) Message() []byte {
	data := &payload{MessageType: "markdown", Markdown: &markdown{Content: strings.Join(msg.lines, "\n")}}
	return data.Build()
}

// NewMarkdown create a markdown message from lines
func NewMarkdown(lines ...interface{}) (*Markdown, error) {
	var markdown Markdown
	for _, line := range lines {
		switch line.(type) {
		case md.Segment:
			if err := markdown.AddSegmentLine(line.(md.Segment)); err != nil {
				return nil, err
			}
		case string:
			if err := markdown.AddLine(line.(string)); err != nil {
				return nil, err
			}
		default:
			if err := markdown.AddLine(fmt.Sprintf("%v", line)); err != nil {
				return nil, err
			}
		}
	}

	return &markdown, nil
}

// Image represents an image
type Image struct {
	len  int
	data []byte
}

// From read image data from reader
func (img *Image) From(reader io.Reader) error {
	// alloc 128k default, double each times
	img.data = make([]byte, MaxImageFileSize/16)
	for len(img.data) <= MaxImageFileSize {
		rn, err := reader.Read(img.data[img.len:])
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		img.len += rn
		if img.len == len(img.data) {
			if 2*img.len > MaxImageFileSize {
				break
			}

			grow := make([]byte, len(img.data)*2)
			copy(grow[0:], img.data[:img.len])
			img.data = grow
		}
	}

	img.len = 0
	img.data = nil // release
	return ErrImageTooLarge
}

// Message implement Messager and build message with image content
func (img *Image) Message() []byte {
	h := md5.New()
	h.Write(img.data[:img.len])

	data := payload{
		MessageType: "image",
		Image: &image{
			Data: base64.StdEncoding.EncodeToString(img.data[:img.len]),
			Hash: fmt.Sprintf("%x", h.Sum(nil)),
		},
	}
	return data.Build()
}

// NewImage create an image message from reader
func NewImage(reader io.Reader) (*Image, error) {
	var img Image
	return &img, img.From(reader)
}

// Card represents a card message
type Card struct {
	articles []*Article
}

// AddArticle add article into card
func (c *Card) AddArticle(article *Article) error {
	if len(c.articles) >= MaxArticleCount {
		return ErrTooManyArticle
	}

	if article.Title == "" || article.Link == "" {
		return errors.New("required article title and link")
	}

	c.articles = append(c.articles, article)
	return nil
}

// Message implement Messager and build message with card content
func (c *Card) Message() []byte {
	data := payload{MessageType: "news", Card: &card{Articles: c.articles}}
	return data.Build()
}

// NewCard create a article card message
func NewCard(articles ...*Article) (*Card, error) {
	var card Card
	for _, article := range articles {
		if err := card.AddArticle(article); err != nil {
			return nil, err
		}
	}

	return &card, nil
}

// Media represents a file message
type Media struct {
	mediaId string
}

// Message implement Messager and build message with media content
func (m *Media) Message() []byte {
	data := payload{MessageType: "file", Media: &media{MediaId: m.mediaId}}
	return data.Build()
}

// NewMedia create a media message from io.Reader
func NewMedia(mediaId string) *Media {
	return &Media{mediaId: mediaId}
}

// payload represents a send request payload
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#消息类型及数据格式
type payload struct {
	MessageType string    `json:"msgtype"`
	Text        *text     `json:"text,omitempty"`
	Markdown    *markdown `json:"markdown,omitempty"`
	Image       *image    `json:"image,omitempty"`
	Card        *card     `json:"card"`
	Media       *media    `json:"file"`
}

// Build build payload to json data
func (p *payload) Build() (bs []byte) {
	// ignored error because all data controlled
	bs, _ = json.Marshal(p)
	return
}

// text represents text message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#文本类型
type text struct {
	Content        string   `json:"content"`
	MentionMembers []string `json:"mentioned_list,omitempty"`
	MentionMobiles []string `json:"mentioned_mobile_list,omitempty"`
}

// markdown represents markdown message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#markdown类型
type markdown struct {
	Content string `json:"content"`
}

// image represents an image message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#图片类型
type image struct {
	Data string `json:"base64"`
	Hash string `json:"md5"`
}

// Article represents an article message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#图文类型
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"url"`
	ImageUrl    string `json:"picurl"`
}

// card represents an article group message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#图文类型
type card struct {
	Articles []*Article `json:"articles"`
}

// media represents a file message data
// see https://work.weixin.qq.com/api/doc/90000/90136/91770#文件类型
type media struct {
	MediaId string `json:"media_id"`
}
