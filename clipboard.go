// Copyright 2013 @atotto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package clipboard_go

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
	// "bufio"
	"golang.org/x/image/bmp"
	"image/jpeg"
	// "github.com/anthonynsimon/bild/imgio"
)

const (
	cfBitmap      = 2
	cfDib         = 8
	cfUnicodetext = 13
	cfDibV5       = 17
	gmemFixed     = 0x0000
)

type fileHeader struct {
	bfType      uint16
	bfSize      uint32
	bfReserved1 uint16
	bfReserved2 uint16
	bfOffBits   uint32
}

type infoHeader struct {
	iSize          uint32
	iWidth         uint32
	iHeight        uint32
	iPLanes        uint16
	iBitCount      uint16
	iCompression   uint32
	iSizeImage     uint32
	iXPelsPerMeter uint32
	iYPelsPerMeter uint32
	iClrUsed       uint32
	iClrImportant  uint32
}

var (
	user32                     = syscall.MustLoadDLL("user32")
	openClipboard              = user32.MustFindProc("OpenClipboard")
	closeClipboard             = user32.MustFindProc("CloseClipboard")
	emptyClipboard             = user32.MustFindProc("EmptyClipboard")
	getClipboardData           = user32.MustFindProc("GetClipboardData")
	setClipboardData           = user32.MustFindProc("SetClipboardData")
	isClipboardFormatAvailable = user32.MustFindProc("IsClipboardFormatAvailable")

	kernel32     = syscall.NewLazyDLL("kernel32")
	globalAlloc  = kernel32.NewProc("GlobalAlloc")
	globalFree   = kernel32.NewProc("GlobalFree")
	globalLock   = kernel32.NewProc("GlobalLock")
	globalUnlock = kernel32.NewProc("GlobalUnlock")
	lstrcpy      = kernel32.NewProc("lstrcpyW")
	copyMemory   = kernel32.NewProc("CopyMemory")
)

func copyInfoHdr(dst *byte, psrc *infoHeader) (string, error) {

	pdst := (*infoHeader)(unsafe.Pointer(dst))

	pdst.iSize = psrc.iSize
	pdst.iWidth = psrc.iWidth
	pdst.iHeight = psrc.iHeight
	pdst.iPLanes = psrc.iPLanes
	pdst.iBitCount = psrc.iBitCount
	pdst.iCompression = psrc.iCompression
	pdst.iSizeImage = psrc.iSizeImage
	pdst.iXPelsPerMeter = psrc.iXPelsPerMeter
	pdst.iYPelsPerMeter = psrc.iYPelsPerMeter
	pdst.iClrUsed = psrc.iClrUsed
	pdst.iClrImportant = psrc.iClrImportant

	return "copy infoHeader success", nil
}

func readUint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func readUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func readClipboard(filename string) (string, error) {
	const (
		fileHeaderLen = 14
		infoHeaderLen = 40
	)

	r, _, err := openClipboard.Call(0)
	if r == 0 {
		return "openClipboard", err
	}
	defer closeClipboard.Call()

	r, _, err = isClipboardFormatAvailable.Call(cfDib)
	if r == 0 {
		return "not Dib format", err
	}

	h, _, err := getClipboardData.Call(cfDib)
	if r == 0 {
		return "getClipboardData", err
	}

	pdata, _, err := globalLock.Call(h)
	if pdata == 0 {
		return "writeToFile globalLock failed!", err
	}

	// var bin_buf bytes.Buffer
	// hmif := new(infoHeader)
	h2 := (*infoHeader)(unsafe.Pointer(pdata))

	fmt.Println(h2)
	dataSize := h2.iSizeImage + fileHeaderLen + infoHeaderLen

	if h2.iSizeImage == 0 && h2.iCompression == 0 {
		iSizeImage := h2.iHeight * ((h2.iWidth*uint32(h2.iBitCount)/8 + 3) &^ 3)
		dataSize += iSizeImage
	}
	log.Println("datasize: ", dataSize, h2.iHeight*((h2.iWidth*uint32(h2.iBitCount)/8+3)&^3))
	// data := make([]byte, dataSize)

	// var hdr *fileHeader
	data := new(bytes.Buffer)
	// hdr := (*bytes.Buffer)(unsafe.Pointer(&data[0]))
	binary.Write(data, binary.LittleEndian, uint16('B')|(uint16('M')<<8))
	binary.Write(data, binary.LittleEndian, uint32(dataSize))
	binary.Write(data, binary.LittleEndian, uint32(0))
	const sizeof_colorbar = 0
	binary.Write(data, binary.LittleEndian, uint32(fileHeaderLen+infoHeaderLen+sizeof_colorbar))
	log.Println("fileHeader ", data.Bytes(), len(data.Bytes()))

	// log.Println("header: ", hdr, data[:8])
	// log.Print("bfOffBits ", hdr.bfOffBits)
	// copyInfoHdr(&data[fileHeaderLen], h2)
	j := 0
	for i := fileHeaderLen; i < int(dataSize); i++ {
		binary.Write(data, binary.BigEndian, *(*byte)(unsafe.Pointer(pdata + uintptr(j))))
		j++
	}

	for i := 0; i < 12; i++ {
		// binary.Write(data, binary.BigEndian, byte(0))
	}

	//	fmt.Println(data.Bytes()[:60])
	// fmt.Println(data.Bytes()[1196900:])

	// // imgio.Save("goimg.png", data, imgio.PNG)
	saveAs(data, "testjpg.jpg")

	return "success", nil
}

func saveAs(dat *bytes.Buffer, filename string) (string, error) {
	// var buf bytes.Buffer
	// buf.Write(dat)
	// f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("try decode")
	original_image, err := bmp.Decode(dat)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("decode success")

	err = jpeg.Encode(f, original_image, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("write file %s success", filename)
	return "succ", nil
}

func writeAll(text string) error {
	r, _, err := openClipboard.Call(0)
	if r == 0 {
		return err
	}
	defer closeClipboard.Call()

	r, _, err = emptyClipboard.Call(0)
	if r == 0 {
		return err
	}

	data := syscall.StringToUTF16(text)

	h, _, err := globalAlloc.Call(gmemFixed, uintptr(len(data)*int(unsafe.Sizeof(data[0]))))
	if h == 0 {
		return err
	}

	l, _, err := globalLock.Call(h)
	if l == 0 {
		return err
	}

	r, _, err = lstrcpy.Call(l, uintptr(unsafe.Pointer(&data[0])))
	if r == 0 {
		return err
	}

	r, _, err = globalUnlock.Call(h)
	if r == 0 {
		return err
	}

	r, _, err = setClipboardData.Call(cfUnicodetext, h)
	if r == 0 {
		return err
	}
	return nil
}
