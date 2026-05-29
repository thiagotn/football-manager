package unit_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"

	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

func TestProcessAvatarWebP(t *testing.T) {
	// PNG de teste 120×80 (não-quadrado, para exercitar o crop)
	src := image.NewRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 120; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, src))

	out, err := services.ProcessAvatarWebP(pngBuf.Bytes())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(out), 12)

	// Magic bytes WebP: "RIFF"...."WEBP"
	assert.Equal(t, "RIFF", string(out[0:4]))
	assert.Equal(t, "WEBP", string(out[8:12]))

	// Decodifica de volta → deve ser 256×256 (crop quadrado + resize)
	img, err := webp.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 256, img.Bounds().Dx())
	assert.Equal(t, 256, img.Bounds().Dy())
}

func TestProcessAvatarWebP_InvalidImage(t *testing.T) {
	_, err := services.ProcessAvatarWebP([]byte("definitely not an image"))
	assert.Error(t, err)
}
