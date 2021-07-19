package main

import (
	"bytes"
	"fmt"
	"github.com/gonutz/d3d9"
	"github.com/icza/mjpeg"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"sync"
	"time"
	"unsafe"
)

const fps int = 60
const timespan int = 10

var imgBuff []image.Image

//var jpgBuff [][]byte

func main() {
	var finish = false
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go Record(fps, timespan, &finish, wg)
	fmt.Print("Press enter to stop recording...")
	fmt.Scanln()
	finish = true
	wg.Wait()

	fmt.Println("Attempting to create video file...")

	aw, err := mjpeg.New("test.avi", 1920, 1080, int32(fps))
	if err != nil {
		fmt.Println("Failed to create file.")
	}

	for i := 0; i < len(imgBuff); i++ {
		frm := Encode(imgBuff[i], false, 80)
		aw.AddFrame(frm)
		fmt.Printf("Added frame %d to file.\n", i)
	}
	aw.Close()
}

/*func task(dur int, wg *sync.WaitGroup) {
	defer wg.Done()
	var i int
	fpsDuration := time.Duration(fps)
	for range time.Tick(time.Second / fpsDuration) {
		i++
		// Capture the screen
		img, err := screenshot.CaptureScreen()
		if err != nil {
			fmt.Println("Failed to capture screen.")
		}
		fmt.Printf("Grabbed frame %d!\n", i)
		// Add image to buffer
		var jpgBytes []byte
		if i%15 == 0 {
			jpgBytes = Encode(img, true, 80)
		} else {
			jpgBytes = Encode(img, false, 80)
		}
		jpgBuff[i] = jpgBytes
		// Check if required time has been reached
		if i/fps >= dur {
			fmt.Printf("%d seconds have passed, Quitting..\n", i/fps)
			break
		}
	}
}
*/
func Record(fps int, timespan int, finish *bool, wg *sync.WaitGroup) {
	defer wg.Done()
	var i int = 1
	fpsDuration := time.Duration(fps)
	mode, device, surface := InitD3D9()
	for range time.Tick(time.Second / fpsDuration) {
		// Capture the screen
		img := CaptureScreen(mode, device, &surface)
		if false {
			fmt.Println("Failed to capture screen.")
			break
		}
		//fmt.Printf("Grabbed frame %d!\n", i/fps)
		// Encode image with JPEG
		//jpgBytes := Encode(img, false, 20)
		if len(imgBuff) >= timespan*fps {
			imgBuff = imgBuff[1:len(imgBuff)]
			imgBuff = append(imgBuff, img)
		} else {
			// Add encoded image to the buffer
			imgBuff = append(imgBuff, img)
		}
		i++
		if *finish {
			break
		}
		/*

			if len(jpgBuff) >= timespan * fps {
				jpgBuff = jpgBuff[1:len(jpgBuff)]
				jpgBuff = append(jpgBuff, jpgBytes)
			} else {
				// Add encoded image to the buffer
				jpgBuff = append(jpgBuff, jpgBytes)
			}

		*/
	}

}

func CaptureScreen(mode d3d9.DISPLAYMODE, device d3d9.Device, surface *d3d9.Surface) image.Image {
	startTime := time.Now()

	device.GetFrontBufferData(0, surface)

	r, _ := surface.LockRect(nil, 0)
	defer surface.UnlockRect()

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
	fmt.Printf("Took %d miliseconds to execute\n", time.Since(startTime).Milliseconds())
	return img
}

func InitD3D9() (d3d9.DISPLAYMODE, d3d9.Device, d3d9.Surface) {
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
	device, _, err := d3d.CreateDevice(
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

	// Create offscreen plain surface
	surface, err := device.CreateOffscreenPlainSurface(
		uint(mode.Width),
		uint(mode.Height),
		d3d9.FMT_A8R8G8B8,
		d3d9.POOL_SYSTEMMEM,
		0,
	)
	//defer surface.Release()

	return mode, *device, *surface
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
