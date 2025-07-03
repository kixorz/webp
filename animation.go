// Copyright 2025 <git@adamkonrad.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webp

// #include "webp.h"
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"errors"
	"image"
	"io"
	"unsafe"
)

// Constants for animation disposal and blending modes.
const (
	// DisposeModeNone indicates that the current frame should remain visible
	// when rendering the next frame.
	DisposeModeNone = 0

	// DisposeModeBackground indicates that the area used by the current frame
	// should be cleared to the background color before rendering the next frame.
	DisposeModeBackground = 1

	// BlendModeBlend indicates that the current frame should be blended with
	// the previous canvas.
	BlendModeBlend = 0

	// BlendModeNoBlend indicates that the current frame should replace the
	// corresponding area in the previous canvas.
	BlendModeNoBlend = 1
)

// AnimationEncoder encodes animated WebP images.
// It provides methods for adding frames, setting animation parameters,
// and encoding the final animation.
//
// Usage:
//
//	enc := webp.NewAnimationEncoder()
//	defer enc.Close()
//
//	// Add frames and set parameters
//	enc.AddFrame(frame1)
//	enc.AddFrame(frame2)
//	enc.SetAnimationParams(params)
//
//	// Encode the animation
//	enc.Encode(outputFile)
type AnimationEncoder struct {
	mux *C.WebPMux
}

// AnimationParams contains parameters for an animated WebP image.
type AnimationParams struct {
	// BackgroundColor is the background color of the canvas stored as ARGB: 0xAARRGGBB
	// For example, 0xFFFFFFFF for white, 0xFF000000 for black, 0x00000000 for transparent.
	BackgroundColor uint32

	// LoopCount is the number of times to repeat the animation.
	// 0 means infinite loop.
	LoopCount int
}

// Frame represents a single frame in an animated WebP image.
type Frame struct {
	// Image is the image data for this frame.
	// It can be any image.Image implementation, but *image.RGBA is recommended
	// for best performance.
	Image image.Image

	// X is the x-offset of the frame within the canvas.
	// The WebP format requires even offsets, so odd values will be rounded down.
	X int

	// Y is the y-offset of the frame within the canvas.
	// The WebP format requires even offsets, so odd values will be rounded down.
	Y int

	// Duration is the display duration of the frame in milliseconds.
	Duration int

	// DisposeMode determines how the area used by the current frame is treated
	// before rendering the next frame. Use DisposeModeNone or DisposeModeBackground.
	DisposeMode int

	// BlendMode determines how transparent pixels of the current frame are blended
	// with those of the previous canvas. Use BlendModeBlend or BlendModeNoBlend.
	BlendMode int
}

// NewAnimationEncoder creates a new AnimationEncoder.
// The returned encoder must be closed with Close() when no longer needed
// to avoid memory leaks.
func NewAnimationEncoder() *AnimationEncoder {
	return &AnimationEncoder{
		mux: webpAnimCreate(),
	}
}

// AddFrame adds a frame to the animation.
//
// The frame's image is encoded as a WebP image and added to the animation.
// Frames are displayed in the order they are added, with the specified duration,
// position, and blending options.
//
// Returns an error if the encoder is closed or if the frame cannot be added.
func (enc *AnimationEncoder) AddFrame(frame Frame) error {
	if enc.mux == nil {
		return errors.New("animation encoder is closed")
	}

	// Encode the image to WebP
	var data []byte
	var err error
	if m, ok := frame.Image.(*image.RGBA); ok {
		data, err = EncodeRGBA(m, 90)
	} else {
		data, err = EncodeRGBA(toRGBAImage(frame.Image), 90)
	}
	if err != nil {
		return err
	}

	// Create a WebPMuxFrameInfo structure
	frameInfo, cData := webpMuxFrameInfoCreate(data, frame.X, frame.Y, frame.Duration, frame.DisposeMode, frame.BlendMode)
	defer C.free(cData)

	// Add the frame to the mux
	if webpAnimPushFrame(enc.mux, &frameInfo, 1) != 1 {
		return errors.New("failed to add frame to animation")
	}

	return nil
}

// SetAnimationParams sets the animation parameters.
//
// This should be called before adding frames to set the background color and
// loop count for the animation.
//
// Returns an error if the encoder is closed or if the parameters cannot be set.
func (enc *AnimationEncoder) SetAnimationParams(params AnimationParams) error {
	if enc.mux == nil {
		return errors.New("animation encoder is closed")
	}

	// Create a WebPMuxAnimParams structure
	animParams := webpMuxAnimParamsCreate(params.BackgroundColor, params.LoopCount)

	// Set the animation parameters
	if webpAnimSetAnimationParams(enc.mux, &animParams) != 1 {
		return errors.New("failed to set animation parameters")
	}

	return nil
}

// Encode assembles the animation and writes it to the given writer.
//
// This should be called after adding all frames and setting animation parameters.
// The resulting WebP file can be viewed in any WebP-compatible viewer that
// supports animation.
//
// Returns an error if the encoder is closed or if the animation cannot be encoded.
func (enc *AnimationEncoder) Encode(w io.Writer) error {
	if enc.mux == nil {
		return errors.New("animation encoder is closed")
	}

	// Assemble the animation
	var webpData C.WebPData
	if webpAnimAssemble(enc.mux, &webpData) != 1 {
		return errors.New("failed to assemble animation")
	}
	defer C.free(unsafe.Pointer(webpData.bytes))

	// Write the data to the writer
	data := webpDataToBytes(webpData)
	_, err := w.Write(data)
	return err
}

// EncodeAnimation encodes an animated WebP image with the given frames and parameters.
//
// This is a convenience function that creates an AnimationEncoder, adds the frames,
// sets the parameters, encodes the animation, and closes the encoder.
//
// Example:
//
//	frames := []webp.Frame{frame1, frame2}
//	params := webp.AnimationParams{BackgroundColor: 0xFFFFFFFF, LoopCount: 0}
//	err := webp.EncodeAnimation(outputFile, frames, params)
func EncodeAnimation(w io.Writer, frames []Frame, params AnimationParams) error {
	enc := NewAnimationEncoder()
	defer enc.Close()

	// Set animation parameters
	if err := enc.SetAnimationParams(params); err != nil {
		return err
	}

	// Add frames
	for _, frame := range frames {
		if err := enc.AddFrame(frame); err != nil {
			return err
		}
	}

	// Encode the animation
	return enc.Encode(w)
}

// EncodeAnimationToBytes encodes an animated WebP image with the given frames and parameters
// and returns the bytes.
//
// This is a convenience function similar to EncodeAnimation but returns the encoded
// animation as a byte slice instead of writing it to a writer.
func EncodeAnimationToBytes(frames []Frame, params AnimationParams) ([]byte, error) {
	var buf bytes.Buffer
	if err := EncodeAnimation(&buf, frames, params); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Close releases resources used by the AnimationEncoder.
//
// This method should be called when the encoder is no longer needed to avoid
// memory leaks. After calling Close, the encoder cannot be used anymore.
func (enc *AnimationEncoder) Close() {
	if enc.mux != nil {
		webpAnimDelete(enc.mux)
		enc.mux = nil
	}
}
