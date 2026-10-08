package printing

import (
	"bytes"
	_ "embed"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Noto Sans covers Vietnamese; embedded, so a print needs no Internet. Licence: fonts/OFL.txt.
var (
	//go:embed fonts/NotoSans-Regular.ttf
	regular []byte
	//go:embed fonts/NotoSans-Bold.ttf
	bold []byte
)

// A4 in points, with the margin kept on every side.
const (
	pageW, pageH = 595.28, 841.89
	margin       = 56.0
	width        = pageW - 2*margin
	textSize     = 10.5
	labelWidth   = 150.0
)

// Page draws a print, top to bottom, starting a new page when one is full. Drawing
// errors are kept and returned once the print is done.
type Page struct {
	pdf       *gopdf.GoPdf
	locale    string
	watermark string
	y         float64
	err       error
}

func newPage(locale, watermark string) (*Page, error) {
	p := &Page{pdf: &gopdf.GoPdf{}, locale: locale, watermark: watermark}
	p.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := p.pdf.AddTTFFontData("regular", regular); err != nil {
		return nil, err
	}
	return p, p.pdf.AddTTFFontData("bold", bold)
}

// next starts a page, its watermark drawn first so the content lies over it.
func (p *Page) next() {
	p.pdf.AddPage()
	p.y = margin
	if p.watermark == "" {
		return
	}
	p.font("bold", 72)
	w, err := p.pdf.MeasureTextWidth(p.watermark)
	p.keep(err)
	p.pdf.SetTextColor(215, 215, 215)
	p.pdf.Rotate(45, pageW/2, pageH/2)
	p.pdf.SetXY((pageW-w)/2, pageH/2-36)
	p.keep(p.pdf.Cell(nil, p.watermark))
	p.pdf.RotateReset()
	p.pdf.SetTextColor(0, 0, 0)
}

func (p *Page) bytes() ([]byte, error) {
	if p.err != nil {
		return nil, p.err
	}
	var b bytes.Buffer
	_, err := p.pdf.WriteTo(&b)
	return b.Bytes(), err
}

func (p *Page) keep(err error) {
	if p.err == nil {
		p.err = err
	}
}

func (p *Page) font(style string, size float64) { p.keep(p.pdf.SetFont(style, "", size)) }

// room starts a new page unless h more points fit on this one.
func (p *Page) room(h float64) {
	if p.y+h > pageH-margin {
		p.next()
	}
}

// lines wraps text to w points, keeping its own line breaks.
func (p *Page) lines(text string, w float64) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		l, err := p.pdf.SplitTextWithWordWrap(para, w)
		p.keep(err)
		out = append(out, l...)
	}
	return out
}

func (p *Page) write(x float64, text string, align int, w float64) {
	p.pdf.SetXY(x, p.y)
	p.keep(p.pdf.CellWithOption(&gopdf.Rect{W: w, H: textSize * 1.4}, text, gopdf.CellOption{Align: align}))
}

// T translates key in the print's locale.
func (p *Page) T(key string, params map[string]any) string {
	return platform.Translate(p.locale, key, params)
}

// Title is the centred title of the document.
func (p *Page) Title(text string) {
	p.font("bold", 15)
	for _, l := range p.lines(text, width) {
		p.room(22)
		p.write(margin, l, gopdf.Center, width)
		p.y += 22
	}
	p.y += 4
}

// Center is a centred line under the title, e.g. the document number.
func (p *Page) Center(text string) {
	p.font("regular", textSize)
	p.room(16)
	p.write(margin, text, gopdf.Center, width)
	p.y += 16
}

// Heading starts a section.
func (p *Page) Heading(text string) {
	p.y += 8
	p.font("bold", 11)
	p.room(18)
	p.write(margin, text, gopdf.Left, width)
	p.y += 18
}

// Text is a paragraph, wrapped to the page.
func (p *Page) Text(text string) {
	p.font("regular", textSize)
	for _, l := range p.lines(text, width) {
		p.room(15)
		p.write(margin, l, gopdf.Left, width)
		p.y += 15
	}
}

// Field is a label and its value on one row; a long value wraps under itself.
func (p *Page) Field(label, value string) {
	p.font("regular", textSize)
	vals := p.lines(value, width-labelWidth)
	p.room(15 * float64(max(1, len(vals))))
	p.write(margin, label, gopdf.Left, labelWidth)
	for _, v := range vals {
		p.write(margin+labelWidth, v, gopdf.Left, width-labelWidth)
		p.y += 15
	}
}

// Column is one column of a table: its share of the page width, and whether its
// cells (amounts) are right-aligned.
type Column struct {
	Title string
	Share float64
	Right bool
}

// Table draws a header row, unless no column has a title, then one row per item; the last
// row may be a total.
func (p *Page) Table(cols []Column, rows [][]string, total bool) {
	row := func(cells []string, style string) {
		p.font(style, textSize)
		p.room(17)
		x := margin
		for i, c := range cols {
			w := width * c.Share
			align := gopdf.Left
			if c.Right {
				align = gopdf.Right
			}
			p.write(x+3, cells[i], align|gopdf.Middle, w-6)
			x += w
		}
		p.y += 17
		p.pdf.SetLineWidth(0.5)
		p.pdf.SetStrokeColor(160, 160, 160)
		p.pdf.Line(margin, p.y, margin+width, p.y)
	}
	p.y += 4
	titles := make([]string, len(cols))
	for i, c := range cols {
		titles[i] = c.Title
	}
	if strings.Join(titles, "") != "" {
		row(titles, "bold")
	}
	for i, r := range rows {
		style := "regular"
		if total && i == len(rows)-1 {
			style = "bold"
		}
		row(r, style)
	}
	p.y += 6
}

// Gap leaves a blank line.
func (p *Page) Gap() { p.y += 10 }

// Money formats an amount in đồng, grouped as the locale writes it.
func (p *Page) Money(v int64) string {
	sep := "."
	if p.locale == "en" {
		sep = ","
	}
	s := strconv.FormatInt(v, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// Decimal writes a decimal number ("21.5") as the locale does.
func (p *Page) Decimal(s string) string {
	if p.locale == "en" {
		return s
	}
	return strings.ReplaceAll(s, ".", ",")
}

// Month writes the month of a business date (YYYY-MM-DD), e.g. 03/2026.
func (p *Page) Month(s string) string {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return s
	}
	if p.locale == "en" {
		return d.Format("January 2006")
	}
	return d.Format("01/2006")
}

// Date formats a business date (YYYY-MM-DD) as the locale writes it; "" stays "".
func (p *Page) Date(s string) string {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return s
	}
	if p.locale == "en" {
		return d.Format("2 Jan 2006")
	}
	return d.Format("02/01/2006")
}
