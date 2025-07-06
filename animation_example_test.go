// Copyright 2025 <git@adamkonrad.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webp

import (
	"image"
	"image/color"
	"image/draw"
	"os"
)

// ExampleEncodeAnimation demonstrates how to create an animated WebP image.
func Example_encodeAnimation() {
	// Create two frames with different colors
	frame1 := createImage(320, 240, color.RGBA{255, 0, 0, 255}) // Red
	frame2 := createImage(320, 240, color.RGBA{0, 0, 255, 255}) // Blue

	// Create frames for the animation
	frames := []Frame{
		{
			Image:       frame1,
			Duration:    1000, // 1 second
			DisposeMode: DisposeModeBackground,
			BlendMode:   BlendModeNoBlend,
		},
		{
			Image:       frame2,
			Duration:    1000, // 1 second
			DisposeMode: DisposeModeBackground,
			BlendMode:   BlendModeNoBlend,
		},
	}

	// Set animation parameters
	params := AnimationParams{
		BackgroundColor: 0xFFFFFFFF, // White background
		LoopCount:       0,          // Infinite loop
	}

	// Create a file to save the animation
	f, err := os.Create("./testdata/animation.webp")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Encode the animation
	if err := EncodeAnimation(f, frames, params); err != nil {
		panic(err)
	}

	// Output:
}

// Helper function to create a solid color image
func createImage(width, height int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}
