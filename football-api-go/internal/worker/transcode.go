// Package worker implements the video transcoding worker: it claims uploaded
// match videos from Postgres, normalizes them to vertical H.264/AAC MP4 via
// ffmpeg and publishes the result (plus a poster frame) to R2.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// ProbeResult carries the ffprobe facts the pipeline decides on.
type ProbeResult struct {
	DurationSeconds float64
	HasVideoStream  bool
}

// Transcoder abstracts the ffmpeg/ffprobe binaries so the state machine can be
// unit-tested without media tooling installed.
type Transcoder interface {
	Probe(ctx context.Context, path string) (ProbeResult, error)
	Transcode(ctx context.Context, inPath, outPath string) error
	Poster(ctx context.Context, videoPath, posterPath string) error
}

// FFmpegTranscoder shells out to ffmpeg/ffprobe (present in the worker image).
type FFmpegTranscoder struct{}

// probeOutput mirrors the subset of `ffprobe -print_format json` we consume.
type probeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// ParseProbeOutput parses ffprobe JSON output into a ProbeResult.
func ParseProbeOutput(data []byte) (ProbeResult, error) {
	var out probeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return ProbeResult{}, fmt.Errorf("ffprobe parse: %w", err)
	}
	res := ProbeResult{}
	for _, s := range out.Streams {
		if s.CodecType == "video" {
			res.HasVideoStream = true
		}
	}
	if out.Format.Duration != "" {
		d, err := strconv.ParseFloat(out.Format.Duration, 64)
		if err != nil {
			return ProbeResult{}, fmt.Errorf("ffprobe duration parse: %w", err)
		}
		res.DurationSeconds = d
	}
	return res, nil
}

func (t *FFmpegTranscoder) Probe(ctx context.Context, path string) (ProbeResult, error) {
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format", "-show_streams",
		path,
	).Output()
	if err != nil {
		return ProbeResult{}, fmt.Errorf("ffprobe: %w", err)
	}
	return ParseProbeOutput(out)
}

// Transcode normalizes any phone video (HEVC/.mov included) to a vertical
// 720×1280 H.264/AAC MP4. Center-crop to 9:16; ffmpeg applies the rotation
// metadata by default, so sideways phone captures come out upright.
func (t *FFmpegTranscoder) Transcode(ctx context.Context, inPath, outPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-i", inPath,
		"-t", "65",
		"-vf", "scale=720:1280:force_original_aspect_ratio=increase,crop=720:1280,fps=30,format=yuv420p",
		"-c:v", "libx264", "-profile:v", "high", "-level", "4.0",
		"-preset", "medium", "-crf", "23",
		"-c:a", "aac", "-b:a", "128k", "-ac", "2", "-ar", "44100",
		"-movflags", "+faststart",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg transcode: %w: %s", err, tail(out, 500))
	}
	return nil
}

// Poster extracts a webp frame (~1s in; falls back to the first frame for
// sub-second clips).
func (t *FFmpegTranscoder) Poster(ctx context.Context, videoPath, posterPath string) error {
	var lastErr error
	for _, ss := range []string{"1", "0"} {
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
			"-ss", ss,
			"-i", videoPath,
			"-frames:v", "1",
			"-c:v", "libwebp", "-quality", "80",
			posterPath,
		)
		out, err := cmd.CombinedOutput()
		// -ss past the end can exit 0 with an empty output — treat as failure.
		if err == nil {
			if info, statErr := os.Stat(posterPath); statErr == nil && info.Size() > 0 {
				return nil
			}
			err = fmt.Errorf("empty poster output")
		}
		lastErr = fmt.Errorf("ffmpeg poster (-ss %s): %w: %s", ss, err, tail(out, 500))
	}
	return lastErr
}

func tail(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[len(b)-n:])
}
