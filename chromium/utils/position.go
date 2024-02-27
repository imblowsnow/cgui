package utils

import (
	"fmt"
	"github.com/kbinani/screenshot"
	"image"
)

func GetScreen() image.Rectangle {
	if screenshot.NumActiveDisplays() < 0 {
		return image.Rect(0, 0, 0, 0)
	}
	bounds := screenshot.GetDisplayBounds(0)
	return bounds
}

func GetCenterPosition(width int, height int) (int, int) {
	bounds := GetScreen()
	fmt.Println("屏幕的宽高", bounds.Dx(), bounds.Dy())
	// 计算左上角的位置
	topLeftX := (bounds.Dx() - width) / 2 / 2
	topLeftY := (bounds.Dy() - height) / 2 / 2 / 2
	return topLeftX, topLeftY
}

func GetAutoWidthHeight() (int, int) {
	bounds := GetScreen()
	// 以屏幕的宽度的80%为宽度
	width := int(float64(bounds.Dx()) * 0.5)
	// 以屏幕的高度的80%为高度
	height := int(float64(bounds.Dy()) * 0.7)
	return width, height
}
