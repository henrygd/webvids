package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var CurentConversion = ""

type progressMsg struct {
	percent    float64
	conversion string
}

func Convert(infile string, outfile string, codec string) tea.Cmd {
	ffmpegArgs := ffmpeg.KwArgs{
		"vf": "scale=-1:1080",
		// "r":     "30",
		// "vsync": "vfr",
	}
	ffmpegArgs["c:v"] = codec
	// x265 specific options
	if codec == "libx265" {
		ffmpegArgs["crf"] = "30"
		// ffmpegArgs["preset"] = "slow"
		ffmpegArgs["movflags"] = "faststart"
	}
	// vp9 specific options
	if codec == "libvpx-vp9" {
		ffmpegArgs["crf"] = "34"
		// ffmpegArgs["deadline"] = "realtime"
		// ffmpegArgs["cpu-used"] = "8"
		// ffmpegArgs["threads"] = "8"
	}
	CurentConversion = codec
	convertWithProgress(infile, outfile, ffmpegArgs)
	return func() tea.Msg {
		return progressMsg{percent: 0.001, conversion: CurentConversion}
	}
	// return progres/sMsg(0.001)
}

// convertWithProgress uses the ffmpeg `-progress` option with a unix-domain socket to report progress
func convertWithProgress(inFileName string, outFileName string, ffmpegArgs ffmpeg.KwArgs) {
	a, err := ffmpeg.Probe(inFileName)
	if err != nil {
		panic(err)
	}
	totalDuration, err := probeDuration(a)
	if err != nil {
		panic(err)
	}

	err = ffmpeg.Input(inFileName).
		Output(outFileName, ffmpegArgs).
		GlobalArgs("-progress", "unix://"+TempSock(totalDuration)).
		OverWriteOutput().
		Silent(true).
		Run()
	if err != nil {
		panic(err)
	}
}

func TempSock(totalDuration float64) string {
	// serve
	sockFileName := path.Join(os.TempDir(), fmt.Sprintf("%d_sock", rand.Int()))
	l, err := net.Listen("unix", sockFileName)
	if err != nil {
		panic(err)
	}

	go func() {
		re := regexp.MustCompile(`out_time_ms=(\d+)`)
		fd, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		buf := make([]byte, 16)
		data := ""
		for {
			_, err := fd.Read(buf)
			if err != nil {
				return
			}
			data += string(buf)
			a := re.FindAllStringSubmatch(data, -1)
			cp := 0.00
			if len(a) > 0 && len(a[len(a)-1]) > 0 {
				c, _ := strconv.Atoi(a[len(a)-1][len(a[len(a)-1])-1])
				cp = float64(c) / totalDuration / 1000000
			}
			if strings.Contains(data, "progress=end") {
				cp = 1.00
			}
			// fmt.Println(cp)
			if cp > 0.00 && cp < 1.01 {
				Program.Send(progressMsg(progressMsg{
					percent:    cp,
					conversion: CurentConversion,
				}))
				if cp == 1.00 {
					time.Sleep(time.Second / 2)
					Program.Send(progressMsg(progressMsg{
						percent:    cp,
						conversion: CurentConversion,
					}))
				}
			}
		}
	}()

	return sockFileName
}

type probeFormat struct {
	Duration string `json:"duration"`
}

type probeData struct {
	Format probeFormat `json:"format"`
}

func probeDuration(a string) (float64, error) {
	pd := probeData{}
	err := json.Unmarshal([]byte(a), &pd)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(pd.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}
