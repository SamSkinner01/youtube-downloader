package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/kkdai/youtube/v2"
)

// FileInfo holds metadata about a downloaded file returned by ProbeFile.
type FileInfo struct {
	// Audio
	AudioCodec   string
	AudioBitrate int // kbps
	SampleRate   int // Hz
	Channels     int
	// Video (empty for audio-only files)
	VideoCodec    string
	VideoBitrate  int // kbps
	Width, Height int
	FPS           string
	// Container
	Duration   string
	FileSizeMB float64
	LossyNote  string // advisory about the encoding chain
}

func (f FileInfo) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Size:     %.2f MB\n", f.FileSizeMB)
	fmt.Fprintf(&b, "  Duration: %s\n", f.Duration)
	if f.VideoCodec != "" {
		fmt.Fprintf(&b, "  Video:    %s  %dx%d  %s fps  %d kbps\n",
			f.VideoCodec, f.Width, f.Height, f.FPS, f.VideoBitrate)
	}
	fmt.Fprintf(&b, "  Audio:    %s  %d kbps  %d Hz  %d ch\n",
		f.AudioCodec, f.AudioBitrate, f.SampleRate, f.Channels)
	if f.LossyNote != "" {
		fmt.Fprintf(&b, "  Note:     %s\n", f.LossyNote)
	}
	return b.String()
}

// ProbeFile uses ffprobe to inspect an output file and return its metadata.
func ProbeFile(path, outFormat string) (FileInfo, error) {
	ffprobe := toolBin("ffprobe")
	cmd := exec.Command(ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return FileInfo{}, err
	}

	var probe struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			BitRate    string `json:"bit_rate"`
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			AvgFPS     string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return FileInfo{}, err
	}

	info := FileInfo{}

	sizeBytes, _ := strconv.ParseInt(probe.Format.Size, 10, 64)
	info.FileSizeMB = float64(sizeBytes) / 1024 / 1024

	durSec, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	m := int(durSec) / 60
	s := int(durSec) % 60
	info.Duration = fmt.Sprintf("%d:%02d", m, s)

	for _, s := range probe.Streams {
		br, _ := strconv.Atoi(s.BitRate)
		switch s.CodecType {
		case "audio":
			info.AudioCodec = s.CodecName
			info.AudioBitrate = br / 1000
			info.SampleRate, _ = strconv.Atoi(s.SampleRate)
			info.Channels = s.Channels
		case "video":
			info.VideoCodec = s.CodecName
			info.VideoBitrate = br / 1000
			info.Width = s.Width
			info.Height = s.Height
			info.FPS = simplifyFPS(s.AvgFPS)
		}
	}

	switch outFormat {
	case "mp3":
		info.LossyNote = "MP3 is lossy. YouTube source was already AAC/Opus (~128–160 kbps). No extra quality is recoverable."
	case "mp4":
		info.LossyNote = "Video stream copied losslessly. Audio re-encoded to AAC from YouTube's AAC/Opus source."
	case "wav":
		info.LossyNote = "WAV is uncompressed PCM — no further loss. YouTube source was already AAC/Opus (~128–160 kbps)."
	}

	return info, nil
}

// VideoInfo returns the title and duration for a URL without downloading.
func VideoInfo(url string) (title, duration string, err error) {
	url = normalizeURL(url)
	client := youtube.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		return "", "", err
	}
	return video.Title, video.Duration.String(), nil
}

// Download fetches and converts the video to outFormat ("mp3", "mp4", or "wav").
// progress is called with values 0.0–1.0 and may be nil.
func Download(url, outPath, quality, outFormat string, progress func(float64)) error {
	url = normalizeURL(url)
	switch outFormat {
	case "mp4":
		return downloadMP4(url, outPath, quality, progress)
	case "wav":
		return downloadAudio(url, outPath, quality, "yt2wav-*.tmp", convertToWAV, progress)
	default:
		return downloadAudio(url, outPath, quality, "yt2mp3-*.tmp", convertToMP3, progress)
	}
}

// SanitizeFilename makes a string safe to use as a filename.
func SanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "", "\"", "", "<", "", ">", "", "|", "-",
	)
	name = strings.TrimSpace(replacer.Replace(name))
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func normalizeURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	return url
}

func downloadAudio(url, outPath, quality, tmpPattern string, convert func(string, string) error, progress func(float64)) error {
	client := youtube.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		return err
	}

	format := pickAudioFormat(video.Formats, quality)
	if format == nil {
		return fmt.Errorf("no audio format found")
	}

	stream, size, err := client.GetStream(video, format)
	if err != nil {
		return err
	}
	defer stream.Close()

	tmp, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := streamToFile(stream, tmp, size, progress); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	return convert(tmpPath, outPath)
}

func downloadMP4(url, outPath, quality string, progress func(float64)) error {
	client := youtube.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		return err
	}

	videoFmt := pickVideoFormat(video.Formats, quality)
	if videoFmt == nil {
		return fmt.Errorf("no video format found")
	}
	audioFmt := pickAudioFormat(video.Formats, "best")
	if audioFmt == nil {
		return fmt.Errorf("no audio format found")
	}

	// Download video stream (progress 0–60%)
	vStream, vSize, err := client.GetStream(video, videoFmt)
	if err != nil {
		return err
	}
	defer vStream.Close()

	tmpV, err := os.CreateTemp("", "yt2mp4-video-*.tmp")
	if err != nil {
		return err
	}
	tmpVPath := tmpV.Name()
	defer os.Remove(tmpVPath)

	if err := streamToFile(vStream, tmpV, vSize, func(p float64) {
		if progress != nil {
			progress(p * 0.6)
		}
	}); err != nil {
		tmpV.Close()
		return err
	}
	tmpV.Close()

	// Download audio stream (progress 60–90%)
	aStream, aSize, err := client.GetStream(video, audioFmt)
	if err != nil {
		return err
	}
	defer aStream.Close()

	tmpA, err := os.CreateTemp("", "yt2mp4-audio-*.tmp")
	if err != nil {
		return err
	}
	tmpAPath := tmpA.Name()
	defer os.Remove(tmpAPath)

	if err := streamToFile(aStream, tmpA, aSize, func(p float64) {
		if progress != nil {
			progress(0.6 + p*0.3)
		}
	}); err != nil {
		tmpA.Close()
		return err
	}
	tmpA.Close()

	if progress != nil {
		progress(0.9)
	}
	return muxMP4(tmpVPath, tmpAPath, outPath)
}

func streamToFile(r io.Reader, w io.Writer, total int64, progress func(float64)) error {
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil && total > 0 {
				progress(float64(downloaded) / float64(total))
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func pickAudioFormat(formats youtube.FormatList, quality string) *youtube.Format {
	audio := formats.Type("audio")
	if len(audio) == 0 {
		return nil
	}
	switch quality {
	case "worst":
		pick := &audio[0]
		for i := range audio {
			if audio[i].Bitrate < pick.Bitrate {
				pick = &audio[i]
			}
		}
		return pick
	case "medium":
		audio.Sort()
		return &audio[len(audio)/2]
	default:
		pick := &audio[0]
		for i := range audio {
			if audio[i].Bitrate > pick.Bitrate {
				pick = &audio[i]
			}
		}
		return pick
	}
}

func pickVideoFormat(formats youtube.FormatList, quality string) *youtube.Format {
	video := formats.Type("video/mp4")
	if len(video) == 0 {
		video = formats.Type("video")
	}
	if len(video) == 0 {
		return nil
	}
	switch quality {
	case "worst":
		pick := &video[0]
		for i := range video {
			if video[i].Height < pick.Height {
				pick = &video[i]
			}
		}
		return pick
	case "medium":
		video.Sort()
		return &video[len(video)/2]
	default:
		pick := &video[0]
		for i := range video {
			if video[i].Height > pick.Height {
				pick = &video[i]
			}
		}
		return pick
	}
}

func convertToMP3(inPath, outPath string) error {
	return runFFmpeg(
		"-fflags", "+discardcorrupt+igndts",
		"-i", inPath,
		"-vn", "-acodec", "libmp3lame", "-q:a", "2",
		"-y", outPath,
	)
}

func convertToWAV(inPath, outPath string) error {
	return runFFmpeg(
		"-fflags", "+discardcorrupt+igndts",
		"-i", inPath,
		"-vn", "-acodec", "pcm_s16le", "-ar", "44100",
		"-y", outPath,
	)
}

func muxMP4(videoPath, audioPath, outPath string) error {
	return runFFmpeg(
		"-i", videoPath, "-i", audioPath,
		"-c:v", "copy", "-c:a", "aac",
		"-movflags", "+faststart",
		"-y", outPath,
	)
}

// toolBin returns the path to a bundled tool (e.g. ffmpeg, ffprobe), falling
// back to the system PATH. Appends .exe on Windows automatically.
func toolBin(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe, err := os.Executable()
	if err != nil {
		return name
	}
	bundled := filepath.Join(filepath.Dir(exe), name)
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}
	return name
}

func runFFmpeg(args ...string) error {
	cmd := exec.Command(toolBin("ffmpeg"), append([]string{"-hide_banner", "-loglevel", "error"}, args...)...)
	errOut, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	errBytes, _ := io.ReadAll(errOut)
	if err := cmd.Wait(); err != nil {
		if len(errBytes) > 0 {
			os.Stderr.Write(errBytes)
		}
		return err
	}
	if filtered := filterWarnings(string(errBytes)); filtered != "" {
		os.Stderr.WriteString(filtered)
	}
	return nil
}

func filterWarnings(s string) string {
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "Error parsing Opus packet header") || strings.TrimSpace(line) == "" {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func simplifyFPS(s string) string {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return s
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return s
	}
	return strconv.FormatFloat(num/den, 'f', 2, 64)
}
