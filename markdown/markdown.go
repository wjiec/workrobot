package markdown

import (
	"fmt"
	"math"
	"strings"
)

// Segment represents a markdown segment
type Segment interface {
	fmt.Stringer

	private()
}

// Inline represents a inline string implemented Segment
type Inline string

// String output the inline string
func (i Inline) String() string {
	return string(i)
}

// private implement and avoid redundant interface matched
func (Inline) private() {}

type TitleLevel int

const (
	MaximalTitle TitleLevel = 1
	MediumTitle  TitleLevel = 3
	MinimumTitle TitleLevel = 6
)

// Title create a leveled title, level range from 1 and 6
// and are automatically set to 1 or 6 when out of range
func Title(level TitleLevel, title string) Segment {
	return Inline(strings.Repeat("#", int(math.Min(math.Max(float64(level), 1), 6))) + " " + title)
}

// Link create a link with title
func Link(title, link string) Segment {
	return Inline(fmt.Sprintf("[%s](%s)", title, link))
}

// Bold create a bold string
func Bold(text interface{}) Segment {
	return Inline(fmt.Sprintf("**%s**", text))
}

// Code create a inline code
func Code(code string) Segment {
	return Inline(fmt.Sprintf("`%s`", code))
}

// Quote create a quote text
func Quote(s string) Segment {
	return Inline("> " + strings.Join(strings.Split(s, "\n"), "\n> "))
}

// ColorGreen create a text with green color
func ColorGreen(s interface{}) Segment {
	return Inline(fmt.Sprintf(`<font color="info">%s</font>`, s))
}

// ColorGray create a text with gray color
func ColorGray(s interface{}) Segment {
	return Inline(fmt.Sprintf(`<font color="comment">%s</font>`, s))
}

// ColorRed create a text with red color
func ColorRed(s interface{}) Segment {
	return Inline(fmt.Sprintf(`<font color="warning">%s</font>`, s))
}

// Join concat all elements into a single segment
func Join(sep string, elements ...interface{}) Segment {
	var results []string
	for _, el := range elements {
		switch el.(type) {
		case Segment:
			results = append(results, el.(Segment).String())
		case string:
			results = append(results, el.(string))
		default:
			results = append(results, fmt.Sprintf("%v", el))
		}
	}

	return Inline(strings.Join(results, sep))
}
