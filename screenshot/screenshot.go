package screenshot

import (
	"fmt"
	"image"
	"reflect"
	"syscall"
	"unsafe"
)

func ScreenRect() (image.Rectangle, error) {
	// Get device context of whole screen
	hDC := GetDC(0)
	if hDC == 0 {
		return image.Rectangle{}, fmt.Errorf("Could not Get primary display err:%d\n", GetLastError())
	}
	defer ReleaseDC(0, hDC)
	x := GetDeviceCaps(hDC, HORZRES)
	y := GetDeviceCaps(hDC, VERTRES)
	return image.Rect(0, 0, x, y), nil
}

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	// Get device context of whole screen
	// Source
	hdcSrc := GetDC(0)
	if hdcSrc == 0 {
		return nil, fmt.Errorf("Could not Get primary display err:%d.\n", GetLastError())
	}
	defer ReleaseDC(0, hdcSrc)

	// Create compatible device context
	// Destination
	hdcDst := CreateCompatibleDC(hdcSrc)
	if hdcDst == 0 {
		return nil, fmt.Errorf("Could not Create Compatible DC err:%d.\n", GetLastError())
	}
	defer DeleteDC(hdcDst)

	// Get width and hight
	x, y := rect.Dx(), rect.Dy()

	// Initialize bitmap
	pbmi := BITMAPINFO{}
	pbmi.BmiHeader.BiSize = uint32(reflect.TypeOf(pbmi.BmiHeader).Size())
	pbmi.BmiHeader.BiWidth = int32(x)
	pbmi.BmiHeader.BiHeight = int32(-y)
	pbmi.BmiHeader.BiPlanes = 1
	pbmi.BmiHeader.BiBitCount = 32
	pbmi.BmiHeader.BiCompression = BI_RGB

	// Create pointer to store bits in
	ppvBits := unsafe.Pointer(uintptr(0))

	// Create compatible bitmap
	hBitmap := CreateDIBSection(hdcDst, &pbmi, DIB_RGB_COLORS, &ppvBits, 0, 0)
	if hBitmap == 0 {
		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", GetLastError())
	}
	if hBitmap == InvalidParameter {
		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
	}
	defer DeleteObject(HGDIOBJ(hBitmap))

	// Select object into device context
	obj := SelectObject(hdcDst, HGDIOBJ(hBitmap))
	if obj == 0 {
		return nil, fmt.Errorf("error occurred and the selected object is not a region err:%d.\n", GetLastError())
	}
	if obj == 0xffffffff { //GDI_ERROR
		return nil, fmt.Errorf("GDI_ERROR while calling SelectObject err:%d.\n", GetLastError())
	}
	defer DeleteObject(obj)

	// Perform bit-block transfer from source to destination context
	if !BitBlt(hdcDst, 0, 0, x, y, hdcSrc, rect.Min.X, rect.Min.Y, SRCCOPY) {
		return nil, fmt.Errorf("BitBlt failed err:%d.\n", GetLastError())
	}

	// Initialise slice with ppvBits as data
	var slice []byte
	hdrp := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdrp.Data = uintptr(ppvBits)
	hdrp.Len = x * y * 4
	hdrp.Cap = x * y * 4

	// Make byte array with length of slice
	imageBytes := make([]byte, len(slice))

	for i := 0; i < len(imageBytes); i += 4 {
		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] = slice[i+2], slice[i], slice[i+1], slice[i+3]
	}

	img := &image.RGBA{imageBytes, 4 * x, image.Rect(0, 0, x, y)}
	return img, nil
}

func GetDeviceCaps(hdc HDC, index int) int {
	ret, _, _ := procGetDeviceCaps.Call(
		uintptr(hdc),
		uintptr(index))

	return int(ret)
}

func GetDC(hwnd HWND) HDC {
	ret, _, _ := procGetDC.Call(
		uintptr(hwnd))

	return HDC(ret)
}

func ReleaseDC(hwnd HWND, hDC HDC) bool {
	ret, _, _ := procReleaseDC.Call(
		uintptr(hwnd),
		uintptr(hDC))

	return ret != 0
}

func DeleteDC(hdc HDC) bool {
	ret, _, _ := procDeleteDC.Call(
		uintptr(hdc))

	return ret != 0
}

func GetLastError() uint32 {
	ret, _, _ := procGetLastError.Call()
	return uint32(ret)
}

func BitBlt(hdcDest HDC, nXDest, nYDest, nWidth, nHeight int, hdcSrc HDC, nXSrc, nYSrc int, dwRop uint) bool {
	ret, _, _ := procBitBlt.Call(
		uintptr(hdcDest),
		uintptr(nXDest),
		uintptr(nYDest),
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(hdcSrc),
		uintptr(nXSrc),
		uintptr(nYSrc),
		uintptr(dwRop))

	return ret != 0
}

func SelectObject(hdc HDC, hgdiobj HGDIOBJ) HGDIOBJ {
	ret, _, _ := procSelectObject.Call(
		uintptr(hdc),
		uintptr(hgdiobj))

	if ret == 0 {
		panic("SelectObject failed")
	}

	return HGDIOBJ(ret)
}

func DeleteObject(hObject HGDIOBJ) bool {
	ret, _, _ := procDeleteObject.Call(
		uintptr(hObject))

	return ret != 0
}

func CreateDIBSection(hdc HDC, pbmi *BITMAPINFO, iUsage uint, ppvBits *unsafe.Pointer, hSection HANDLE, dwOffset uint) HBITMAP {
	ret, _, _ := procCreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(pbmi)),
		uintptr(iUsage),
		uintptr(unsafe.Pointer(ppvBits)),
		uintptr(hSection),
		uintptr(dwOffset))

	return HBITMAP(ret)
}

func CreateCompatibleDC(hdc HDC) HDC {
	ret, _, _ := procCreateCompatibleDC.Call(
		uintptr(hdc))

	if ret == 0 {
		panic("Create compatible DC failed")
	}

	return HDC(ret)
}

type (
	HANDLE  uintptr
	HWND    HANDLE
	HGDIOBJ HANDLE
	HDC     HANDLE
	HBITMAP HANDLE
)

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors *RGBQUAD
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type RGBQUAD struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

const (
	HORZRES          = 8
	VERTRES          = 10
	BI_RGB           = 0
	InvalidParameter = 2
	DIB_RGB_COLORS   = 0
	SRCCOPY          = 0x00CC0020
)

var (
	modgdi32               = syscall.NewLazyDLL("gdi32.dll")
	moduser32              = syscall.NewLazyDLL("user32.dll")
	modkernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetDC              = moduser32.NewProc("GetDC")
	procReleaseDC          = moduser32.NewProc("ReleaseDC")
	procDeleteDC           = modgdi32.NewProc("DeleteDC")
	procBitBlt             = modgdi32.NewProc("BitBlt")
	procDeleteObject       = modgdi32.NewProc("DeleteObject")
	procSelectObject       = modgdi32.NewProc("SelectObject")
	procCreateDIBSection   = modgdi32.NewProc("CreateDIBSection")
	procCreateCompatibleDC = modgdi32.NewProc("CreateCompatibleDC")
	procGetDeviceCaps      = modgdi32.NewProc("GetDeviceCaps")
	procGetLastError       = modkernel32.NewProc("GetLastError")
)
