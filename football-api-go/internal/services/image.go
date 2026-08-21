package services

import (
	"bytes"
	"fmt"
	"image/jpeg"  // decoder e encoder JPEG (fotos do feed)
	_ "image/png" // registra decoder PNG

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // registra decoder WebP (entrada)
)

// avatarSize é o lado (px) do avatar quadrado final.
const avatarSize = 256

// Limites da foto do feed de rachão (vertical; downscale apenas, sem upscale).
const (
	feedImageMaxWidth  = 1080
	feedImageMaxHeight = 1920
	feedImageQuality   = 85
)

// ProcessFeedImageJPEG decodifica a foto enviada (JPG/PNG/WebP), corrige a
// orientação EXIF e redimensiona para caber em 1080×1920 preservando a
// proporção (sem upscale), devolvendo JPEG — usado pelo worker de mídia.
// JPEG em vez de WebP: o encoder nativewebp é lossless e ficaria pesado
// demais para fotos grandes.
func ProcessFeedImageJPEG(data []byte) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}
	fitted := imaging.Fit(img, feedImageMaxWidth, feedImageMaxHeight, imaging.Lanczos)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, fitted, &jpeg.Options{Quality: feedImageQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode: %w", err)
	}
	return buf.Bytes(), nil
}

// ProcessAvatarWebP decodifica a imagem enviada (JPG/PNG/WebP), corrige a
// orientação EXIF, faz crop quadrado centralizado + resize para 256×256 e
// devolve os bytes em WebP — paridade com o upload de avatar da API Python (v1).
func ProcessAvatarWebP(data []byte) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	// Fill = crop quadrado centralizado + resize exato para avatarSize×avatarSize.
	square := imaging.Fill(img, avatarSize, avatarSize, imaging.Center, imaging.Lanczos)

	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, square, nil); err != nil {
		return nil, fmt.Errorf("webp encode: %w", err)
	}
	return buf.Bytes(), nil
}
