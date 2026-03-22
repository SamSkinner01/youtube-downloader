package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"youtube-downloader/internal/downloader"

	"github.com/kkdai/youtube/v2"
)

func main() {
	output := flag.String("o", "", "Output file path (default: <title>.<format>)")
	quality := flag.String("q", "best", "Quality: best, medium, worst")
	outFormat := flag.String("f", "mp3", "Output format: mp3, mp4, wav")
	formats := flag.Bool("formats", false, "List all available streams and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: youtube-downloader [flags] <youtube-url>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	url := flag.Arg(0)

	if *outFormat != "mp3" && *outFormat != "mp4" && *outFormat != "wav" {
		fmt.Fprintf(os.Stderr, "error: -f must be mp3, mp4, or wav\n")
		os.Exit(1)
	}

	client := youtube.Client{}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	video, err := client.GetVideo(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get video: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Title:    %s\n", video.Title)
	fmt.Printf("Duration: %s\n", video.Duration)

	if *formats {
		audio := video.Formats.Type("audio")
		fmt.Printf("\nAudio streams (%d):\n", len(audio))
		for _, f := range audio {
			fmt.Printf("  %-40s %d kbps\n", f.MimeType, f.Bitrate/1000)
		}
		vid := video.Formats.Type("video/mp4")
		fmt.Printf("\nVideo streams (%d):\n", len(vid))
		for _, f := range vid {
			fmt.Printf("  %-40s %dp  %d kbps\n", f.MimeType, f.Height, f.Bitrate/1000)
		}
		return
	}

	outPath := *output
	if outPath == "" {
		outPath = downloader.SanitizeFilename(video.Title) + "." + *outFormat
	} else if !strings.HasSuffix(strings.ToLower(outPath), "."+*outFormat) {
		outPath += "." + *outFormat
	}
	outPath, err = filepath.Abs(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Output:   %s\n", outPath)

	err = downloader.Download(url, outPath, *quality, *outFormat, func(p float64) {
		fmt.Printf("\rDownloading... %.1f%%", p*100)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\rDone: %s\n", outPath)

	if info, err := downloader.ProbeFile(outPath, *outFormat); err == nil {
		fmt.Println("\nFile info:")
		fmt.Print(info)
	}
}
