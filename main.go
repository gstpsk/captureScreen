package main

import "C"
import (
	"bytes"
	"captureScreen/screenshot"
	"fmt"
	"github.com/gen2brain/x264-go"
	_ "github.com/gen2brain/x264-go/x264c"
	"github.com/glycerine/rbuf"
	"github.com/gonutz/d3d9"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"sync"
	"time"
	"unsafe"
)

const fps int = 35
const timespan int = 10

var imgBuff []image.Image

//var jpgBuff [][]byte

func main() {
	var finish = false
	wg := new(sync.WaitGroup)
	wg.Add(1)
	var data []byte
	go func() {
		data = Record(fps, &finish, wg)
	}()
	fmt.Print("Press enter to stop recording...")
	fmt.Scanln()
	finish = true
	wg.Wait()

	fmt.Print("Attempting to create video file...")

	err := ioutil.WriteFile("test.264", data, 0644)
	check(err)

	fmt.Println("Done!")
	//exec.Cmd{Path: }
	// resources\ffmpeg.exe -f h264 -i test.264 -c:v copy test.mp4 -y
}

func Record(fps int, finish *bool, wg *sync.WaitGroup) []byte {
	defer wg.Done()
	fmt.Println("Initiating recording...")
	var i int = 1
	captureGroup := new(sync.WaitGroup)
	rect, err := screenshot.ScreenRect()
	check(err)

	// Initialize buffer
	//buf := bytes.NewBuffer(make([]byte, 0))
	buf2 := rbuf.NewFixedSizeRingBuf(50 * 1000 * 1000)

	// Initialize h264 encoder
	opts := &x264.Options{
		Width:     rect.Dx(),
		Height:    rect.Dy(),
		FrameRate: fps,
		Tune:      "zerolatency",
		Preset:    "ultrafast",
		Profile:   "baseline",
		LogLevel:  x264.LogInfo,
	}
	enc, err := x264.NewEncoder(buf2, opts)

	defer enc.Close()
	check(err)

	// Initialize d3d9
	mode, device := InitD3D9()

	// Start recording
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	for range ticker.C {
		captureGroup.Add(1)
		fmt.Printf("Grabbing frame %d took ", i)
		go d3d9Capture(mode, device, captureGroup, enc)
		captureGroup.Wait()
		i++
		if *finish {
			break
		}
	}

	// Return buffer
	enc.Flush()
	return buf2.Bytes()
}

func Capture(rect image.Rectangle, cg *sync.WaitGroup, enc *x264.Encoder) {
	// Capture the screen
	defer cg.Done()
	startTime := time.Now()
	img, err := screenshot.CaptureRect(rect)
	fmt.Printf("%dms\n", time.Since(startTime).Milliseconds())
	check(err)

	err = enc.Encode(img)

	check(err)
}

func d3d9Capture(mode d3d9.DISPLAYMODE, device *d3d9.Device, cg *sync.WaitGroup, enc *x264.Encoder) {
	// Capture the screen
	defer cg.Done()
	startTime := time.Now()
	img := d3d9Screenshot(mode, device)
	fmt.Printf("%dms\n", time.Since(startTime).Milliseconds())

	err := enc.Encode(img)
	check(err)
}

func d3d9Screenshot(mode d3d9.DISPLAYMODE, device *d3d9.Device) image.Image {
	// Create offscreen plain surface
	surface, _ := device.CreateOffscreenPlainSurface(
		uint(mode.Width),
		uint(mode.Height),
		d3d9.FMT_A8R8G8B8,
		d3d9.POOL_SYSTEMMEM,
		0,
	)
	defer surface.Release()
	//fmt.Println("Trying to get front buffer data...")
	err := device.GetFrontBufferData(0, surface)
	check(err)
	//fmt.Println("Got front buffer data")
	r, err := surface.LockRect(nil, 0)
	check(err)
	defer surface.UnlockRect()
	//fmt.Println("Locked rectangle")
	if r.Pitch != int32(mode.Width*4) {
		panic("Weird ass padding bruh")
	}

	// Create image of same size as surface
	img := image.NewRGBA(image.Rect(0, 0, int(mode.Width), int(mode.Height)))
	// Copy the shites
	for i := range img.Pix {
		img.Pix[i] = *((*byte)(unsafe.Pointer(r.PBits + uintptr(i))))
	}
	// Covert ARGB to RGBA
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0], img.Pix[i+2] = img.Pix[i+2], img.Pix[i+0]
	}
	return img
}

func InitD3D9() (d3d9.DISPLAYMODE, *d3d9.Device) {
	d3d, err := d3d9.Create(d3d9.SDK_VERSION)
	//defer d3d.Release()
	if err != nil {
		panic("Failed to bind to d3d9")
	}
	mode, err := d3d.GetAdapterDisplayMode(d3d9.ADAPTER_DEFAULT)

	// Check if display format is known
	if mode.Format != d3d9.FMT_X8R8G8B8 && mode.Format != d3d9.FMT_A8R8G8B8 {
		panic("Unknown display mode format")
	}

	// Create device
	device, _, _ := d3d.CreateDevice(
		d3d9.ADAPTER_DEFAULT,
		d3d9.DEVTYPE_HAL,
		0,
		d3d9.CREATE_SOFTWARE_VERTEXPROCESSING,
		d3d9.PRESENT_PARAMETERS{
			Windowed:         1,
			BackBufferCount:  1,
			BackBufferWidth:  mode.Width,
			BackBufferHeight: mode.Height,
			SwapEffect:       d3d9.SWAPEFFECT_DISCARD,
		},
	)
	//defer device.Release()

	return mode, device
}

// Encode the image using jpeg to make mem happy :)
func Encode(img image.Image, hq bool, q int) []byte {
	if false {
		img = resize.Resize(640, 480, img, resize.Bilinear)
		img = resize.Resize(1920, 1080, img, resize.Bilinear)
	}
	o := jpeg.Options{Quality: q}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, &o)
	return buf.Bytes()
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
