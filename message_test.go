package workrobot

import (
	"strings"
	"testing"

	md "github.com/wjiec/workrobot/markdown"

	"github.com/stretchr/testify/assert"
)

func TestNewMention(t *testing.T) {
	msg := NewMention([]string{"user"}, nil, true)

	data := msg.payload()
	assert.NotNil(t, data.Text)
	assert.Equal(t, data.MessageType, "text")
	assert.Equal(t, msg.members, []string{"user"})
	assert.Equal(t, msg.all, true)
}

func TestNewText(t *testing.T) {
	msg, err := NewText("text message")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "text message", msg.content)
}

func TestNewMarkdown(t *testing.T) {
	msg, err := NewMarkdown(md.Title(md.MaximalTitle, "title"), "plain text", md.ColorGreen("end"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, msg.lines, "# title")
	assert.Contains(t, msg.lines, "plain text")
	assert.Contains(t, msg.lines, `<font color="info">end</font>`)
}

func TestNewImage(t *testing.T) {
	buf := strings.NewReader(strings.Repeat("0", MaxImageFileSize+1))
	_, err := NewImage(buf)

	assert.ErrorIs(t, err, ErrImageTooLarge)
}

func TestNewCard(t *testing.T) {
	_, err := NewCard(&Article{Title: "hello"})

	assert.NotNil(t, err)
}

func TestNewMedia(t *testing.T) {
	msg := NewMedia("media-id")

	assert.Equal(t, "media-id", msg.mediaId)
}
