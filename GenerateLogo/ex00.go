package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for x := 0; x < 300; x++ {
		for y := 0; y < 300; y++ {
			img.Set(x, y, color.RGBA{0, 255, 80, 255})
		}
	}

	for x := 0; x < 300; x++ {
		for y := 0; y < 300; y++ {
			if y == x/3 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
			if x == 5 {
				img.Set(x, y, color.RGBA{195, 20, 60, 255})
				img.Set(x+y, y+x, color.RGBA{195, 20, 255, 255})
				img.Set(x*y, y/x, color.RGBA{195, 20, 60, 255})
			}
			img.Set(x*x, y+x, color.RGBA{195, 20, 255, 255})
			img.Set(x/2, y-x, color.RGBA{0, 0, 255, 255})
			img.Set(x-y, y/2, color.RGBA{255, 0, 255, 255})
			img.Set(x+y, y/3, color.RGBA{200, 50, 255, 255})
			img.Set(x-y, y*6, color.RGBA{20, 50, 255, 255})
			img.Set(x/3*y, y/3*x, color.RGBA{100, 50, 200, 255})
		}
	}

	outputFile, err := os.Create("amazing_logos.png")
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()
	err = png.Encode(outputFile, img)
	if err != nil {
		log.Fatal(err)
	}
}
