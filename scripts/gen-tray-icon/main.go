// Command gen-tray-icon rasterizes web/qxweb/public/brand-icon.svg for the launcher tray.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
)

func main() {
	outDir := "."
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	svgPath := filepath.Join("..", "..", "web", "qxweb", "public", "brand-icon.svg")
	if len(os.Args) > 2 {
		svgPath = os.Args[2]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	images := make(map[int][]byte, len(sizes))
	for _, size := range sizes {
		data, err := renderIcon(svgPath, size)
		if err != nil {
			log.Fatal(err)
		}
		images[size] = data
	}

	if err := os.WriteFile(filepath.Join(outDir, "icon.png"), images[32], 0o644); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "icon-16.png"), images[16], 0o644); err != nil {
		log.Fatal(err)
	}

	ico, err := encodeICO(images)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "icon.ico"), ico, 0o644); err != nil {
		log.Fatal(err)
	}

	log.Printf("wrote tray icons from %s to %s (ico=%d bytes)", svgPath, outDir, len(ico))
}

func renderIcon(svgPath string, size int) ([]byte, error) {
	renderSize := size
	if size <= 32 {
		renderSize = 256
	} else if size <= 64 {
		renderSize = 128
	}
	img, err := rasterizeSVG(svgPath, renderSize)
	if err != nil {
		return nil, err
	}
	drawDesktopOutlined(img, renderSize)
	if renderSize != size {
		img = downscale(img, size)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func rasterizeSVG(svgPath string, size int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIcon(svgPath, oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1)
	return img, nil
}

// drawDesktopOutlined draws Ant Design DesktopOutlined as a filled silhouette (visible at 16px).
func drawDesktopOutlined(dst *image.RGBA, size int) {
	white := color.RGBA{255, 255, 255, 255}
	cx := size / 2

	// Fill most of the tray cell — OS scales to ~16px, so maximize contrast and size.
	iconW := int(float64(size)*0.62 + 0.5)
	if iconW < 6 {
		iconW = 6
	}
	iconH := iconW * 13 / 16
	if iconH < 7 {
		iconH = 7
	}

	top := (size - iconH) / 2
	screenH := iconH * 74 / 100
	if screenH < 4 {
		screenH = 4
	}

	left := cx - iconW/2
	fillRect(dst, left, top, iconW, screenH, white)

	neckW := max(2, iconW/4)
	neckH := max(2, iconH/6)
	fillRect(dst, cx-neckW/2, top+screenH, neckW, neckH, white)

	baseW := max(3, iconW*2/3)
	baseH := max(2, iconH/7)
	fillRect(dst, cx-baseW/2, top+screenH+neckH, baseW, baseH, white)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fillRect(dst *image.RGBA, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(dst, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func downscale(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encodeICO(images map[int][]byte) ([]byte, error) {
	sizes := make([]int, 0, len(images))
	for size := range images {
		sizes = append(sizes, size)
	}
	for i := 0; i < len(sizes); i++ {
		for j := i + 1; j < len(sizes); j++ {
			if sizes[j] < sizes[i] {
				sizes[i], sizes[j] = sizes[j], sizes[i]
			}
		}
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(len(sizes))); err != nil {
		return nil, err
	}

	offset := 6 + 16*len(sizes)
	for _, size := range sizes {
		data := images[size]
		dim := byte(size)
		if size >= 256 {
			dim = 0
		}
		buf.WriteByte(dim)
		buf.WriteByte(dim)
		buf.WriteByte(0)
		buf.WriteByte(0)
		if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint16(32)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint32(len(data))); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint32(offset)); err != nil {
			return nil, err
		}
		offset += len(data)
	}
	for _, size := range sizes {
		if _, err := buf.Write(images[size]); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
