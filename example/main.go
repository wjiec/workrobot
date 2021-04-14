package main

import (
	"net/http"
	"net/url"
	"os"

	"github.com/wjiec/workrobot"
	"github.com/wjiec/workrobot/markdown"
)

func main() {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: func(_ *http.Request) (*url.URL, error) {
				return url.Parse("http://proxy.example.com:8888")
			},
		},
	}

	c, err := workrobot.NewClient("[your work-wx robot key]", workrobot.WithHttpClient(httpClient))
	if err != nil {
		panic(err)
	}

	text, _ := workrobot.NewText("hello world from workrobot")
	text.MentionMember("jayson")
	text.MentionMobile("18012345678")

	md, _ := workrobot.NewMarkdown(markdown.Title(markdown.MediumTitle, "Server Closed"),
		markdown.Quote("Ip: 10.2.3.4\nAction: Restart"),
		markdown.Link("view", "http://dashboard.example.com"),
		markdown.ColorOrangeRed("something message..."))

	f, _ := os.Open("/image.jpg")
	image, _ := workrobot.NewImage(f)

	result, _ := c.Uploader().UploadFromReader(f)
	media := workrobot.NewMedia(result.Id)

	if err := c.Send(text, md, image, media); err != nil {
		panic(err)
	}

	if err := c.SendConcurrency(false, text, md, image, media); err != nil {
		panic(err)
	}
}
