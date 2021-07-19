package main

import (
	"bytes"
	"captureScreen/screenshot"
	"fmt"
	"github.com/icza/mjpeg"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"time"
)

var jpgBuff = make(map[int][]byte)
var fps int = 15

func main() {
	go task(10)

	time.Sleep(time.Second * 11)

	fmt.Println("Attempting to create video file...")

	aw, err := mjpeg.New("test.avi", 1920, 1080, int32(fps))
	if err != nil {
		fmt.Println("Failed to create file.")
	}


	for i := 0; i < len(jpgBuff); i++ {
		aw.AddFrame(jpgBuff[i])
		fmt.Printf("Added frame %d to file.\n", i)
	}
	aw.Close()
}

func task(dur int) {
	var i int
	fpsDuration := time.Duration(fps)
	for range time.Tick(time.Second / fpsDuration){
		i++
		// Capture the screen
		img, err := screenshot.CaptureScreen()
		if err != nil {
			fmt.Println("Failed to capture screen.")
		}
		fmt.Printf("Grabbed frame %d!\n", i)
		// Add image to buffer
		var jpgBytes []byte
		if i % 15 == 0 {
			jpgBytes = Encode(img, true)
		} else {
			jpgBytes = Encode(img, false)
		}
		jpgBuff[i] = jpgBytes
		// Check if required time has been reached
		if i / fps >= dur {
			fmt.Printf("%d seconds have passed, Quitting..\n", i / fps)
			break
		}
	}
}

func Encode(img image.Image, hq bool) []byte {
	if false {
		img = resize.Resize(640, 480, img, resize.Bilinear)
		img = resize.Resize(1920, 1080, img, resize.Bilinear)
	}
	o := jpeg.Options{Quality: 80}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, &o)
	return buf.Bytes()
}
