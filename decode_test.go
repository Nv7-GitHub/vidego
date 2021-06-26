package vidego

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	vid, err := NewDecoder("sample.mp4")
	if err != nil {
		t.Fatal(err)
	}

	frameNum := 0

	var imgs []image.Image
	var cont bool

	for {
		cont, imgs, err = vid.GetNextFrame()
		if err != nil {
			t.Fatal(err)
		}
		if !cont {
			break
		}
		if imgs == nil {
			continue
		}

		for _, im := range imgs {
			f, err := os.Create(fmt.Sprintf("out/%d.png", frameNum))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			err = png.Encode(f, im)
			if err != nil {
				t.Fatal(err)
			}
			frameNum++
		}
	}
}
