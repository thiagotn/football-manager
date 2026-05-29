package services

import (
	"bytes"
	"fmt"
	_ "image/jpeg" // registra decoder JPEG
	_ "image/png"  // registra decoder PNG

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // registra decoder WebP (entrada)
)

// avatarSize é o lado (px) do avatar quadrado final.
const avatarSize = 256

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
