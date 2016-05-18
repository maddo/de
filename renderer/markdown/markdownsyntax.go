package renderer

import (
	"bytes"
	"github.com/driusan/de/demodel"
	//"fmt"
	"github.com/driusan/de/renderer"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"unicode"
	//	"image/color"
	"image/draw"
	"strings"
)

type MarkdownSyntax struct{}

func (rd *MarkdownSyntax) CanRender(buf demodel.CharBuffer) bool {
	return strings.HasSuffix(buf.Filename, ".md") || strings.HasSuffix(buf.Filename, "COMMIT_EDITMSG")
}
func (rd *MarkdownSyntax) calcImageSize(buf demodel.CharBuffer) image.Rectangle {
	metrics := renderer.MonoFontFace.Metrics()
	runes := bytes.Runes(buf.Buffer)
	_, MglyphWidth, _ := renderer.MonoFontFace.GlyphBounds('M')
	rt := image.ZR
	var lineSize fixed.Int26_6
	for _, r := range runes {
		_, glyphWidth, _ := renderer.MonoFontFace.GlyphBounds(r)
		switch r {
		case '\t':
			lineSize += MglyphWidth * 8
		case '\n':
			rt.Max.Y += metrics.Height.Ceil()
			lineSize = 0
		default:
			lineSize += glyphWidth
		}
		if lineSize.Ceil() > rt.Max.X {
			rt.Max.X = lineSize.Ceil()
		}
	}
	rt.Max.Y += metrics.Height.Ceil() + 1
	return rt
}

func (rd *MarkdownSyntax) Render(buf demodel.CharBuffer) (image.Image, renderer.ImageMap, error) {
	dstSize := rd.calcImageSize(buf)
	dst := image.NewRGBA(dstSize)
	metrics := renderer.MonoFontFace.Metrics()
	writer := font.Drawer{
		Dst:  dst,
		Src:  &image.Uniform{renderer.TextColour},
		Dot:  fixed.P(0, metrics.Ascent.Floor()),
		Face: renderer.MonoFontFace,
	}
	runes := bytes.Runes(buf.Buffer)

	im := make(renderer.ImageMap, 0)

	// Used for calculating the size of a tab.
	_, MglyphWidth, _ := renderer.MonoFontFace.GlyphBounds('M')

	var nextColor image.Image
	// the beginning of a file is the start of the first line..
	lineStart := true

	for i, r := range runes {
		// Do this inside the loop anyways, in case someone changes it to a
		// variable width font..
		_, glyphWidth, _ := renderer.MonoFontFace.GlyphBounds(r)
		switch r {
		case '\n':
			lineStart = true
			writer.Src = &image.Uniform{renderer.TextColour}

		default:
			if lineStart {

				switch r {
				case '#':
					// heading
					writer.Src = &image.Uniform{renderer.CommentColour}
				case '*', '-', '+':
					// lists
					if i < len(runes)-1 && unicode.IsSpace(runes[i+1]) {

						writer.Src = &image.Uniform{renderer.KeywordColour}
						nextColor = &image.Uniform{renderer.TextColour}
					}
				default:
					// the \n already reset it, no need to do this.
					//writer.Src = &image.Uniform{renderer.TextColour}
				}

				lineStart = false
			}
		}

		runeRectangle := image.Rectangle{}
		runeRectangle.Min.X = writer.Dot.X.Ceil()
		runeRectangle.Min.Y = writer.Dot.Y.Ceil() - metrics.Ascent.Floor()
		switch r {
		case '\t':
			runeRectangle.Max.X = runeRectangle.Min.X + 8*MglyphWidth.Ceil()
		case '\n':
			runeRectangle.Max.X = dstSize.Max.X
		default:
			runeRectangle.Max.X = runeRectangle.Min.X + glyphWidth.Ceil()
		}
		runeRectangle.Max.Y = runeRectangle.Min.Y + metrics.Height.Ceil() + 1

		im = append(im, renderer.ImageLoc{runeRectangle, uint(i)})
		if uint(i) >= buf.Dot.Start && uint(i) <= buf.Dot.End {
			// it's in dot, so highlight the background
			draw.Draw(
				dst,
				runeRectangle,
				&image.Uniform{renderer.TextHighlight},
				image.ZP,
				draw.Src,
			)
		}

		switch r {
		case '\t':
			writer.Dot.X += glyphWidth * 8
			continue
		case '\n':
			writer.Dot.Y += metrics.Height
			writer.Dot.X = 0
			continue
		}
		writer.DrawString(string(r))

		if nextColor != nil {
			writer.Src = nextColor
			nextColor = nil
		}
	}

	return dst, im, nil
}