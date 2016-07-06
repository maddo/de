package actions

import (
"fmt"
	"os"
	plumblib "9fans.net/go/plumb"
	"9fans.net/go/plan9"
	"sync"
	//"io/ioutil"

	"github.com/driusan/de/demodel"
)

var pMutex sync.Mutex
var pBuff *demodel.CharBuffer
var pView demodel.Viewport

func init() {
	go func() {
		f, err := plumblib.Open("edit", plan9.OREAD)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		var m plumblib.Message
		for {
			fmt.Printf("Waiting for edit message\n")
			m.Recv(f)
			pMutex.Lock()
			fmt.Printf("Received message: %s %s\n", string(m.Type), string(m.Data))
			for a := m.Attr; a != nil; a = a.Next {
				fmt.Printf("Name: %s=%s\n", a.Name, a.Value)
			}
			OpenFile(string(m.Data), pBuff, pView)
			pMutex.Unlock()					
		}
	}()
}
func plumb(content []byte, buff *demodel.CharBuffer, v demodel.Viewport) error {
	pMutex.Lock()
	defer pMutex.Unlock()
	pBuff = buff
	pView = v
	print(string("Plumbing"), string(content))
	fid, err := plumblib.Open("send", plan9.OWRITE)
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	wd, _ := os.Getwd()	
	m := plumblib.Message{
		Src: "de",
		Dst: "",
		Dir: wd,
		Type: "text",
		Data: content,
	}
	return m.Send(fid)
}
func PlumbOrFindNext(From, To demodel.Position, buff *demodel.CharBuffer, v demodel.Viewport) {
	if buff == nil {
		return
	}
	dot := demodel.Dot{}
	i, err := From(*buff)
	if err != nil {
		return
	}
	dot.Start = i

	i, err = To(*buff)
	if err != nil {
		return
	}
	dot.End = i + 1

	word := string(buff.Buffer[dot.Start:dot.End])

	if err := plumb([]byte(word), buff, v); err == nil {
		return
	}

	// the file doesn't exist, so find the next instance of word.
	lenword := dot.End - dot.Start
	for i := dot.End; i < uint(len(buff.Buffer))-lenword; i++ {
		if string(buff.Buffer[i:i+lenword]) == word {
			buff.Dot.Start = i
			buff.Dot.End = i + lenword - 1
			return
		}
	}
}

func TagPlumbOrFindNext(From, To demodel.Position, buff *demodel.CharBuffer, v demodel.Viewport) {
	if buff == nil || buff.Tagline == nil {
		return
	}
	dot := demodel.Dot{}
	i, err := From(*buff)
	if err != nil {
		return
	}
	dot.Start = i

	i, err = To(*buff)
	if err != nil {
		return
	}
	dot.End = i + 1

	// find the word between From and To in the tagline
	word := string(buff.Tagline.Buffer[dot.Start:dot.End])

	if err := plumb([]byte(word), buff, v); err == nil {
		return
	}

	// the file doesn't exist, so find the next instance of word inside
	// the *non-tag* buffer.
	lenword := dot.End - dot.Start
	for i := buff.Dot.End; i < uint(len(buff.Buffer))-lenword; i++ {
		if string(buff.Buffer[i:i+lenword]) == word {
			buff.Dot.Start = i
			buff.Dot.End = i + lenword - 1
			return
		}
	}
}