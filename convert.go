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

var hevcArgs = ffmpeg.KwArgs{
	"c:v": "libx265",
	"vf":  "scale=-1:1080",
	"crf": "30",
	"r":   "30",
}

type progressMsg float64

func Convert() tea.Msg {
	// ConvertWithProgress("adsfsdf.mp4", "output.mp4")
	// ConvertWithProgress("test.mp4", "output.mp4")
	ConvertWithProgress("test2.mp4", "output.mp4")
	// ConvertWithProgress("test3.mp4", "output.mp4")
	return progressMsg(0.001)
}

// ConvertWithProgress uses the ffmpeg `-progress` option with a unix-domain socket to report progress
func ConvertWithProgress(inFileName, outFileName string) {
	a, err := ffmpeg.Probe(inFileName)
	if err != nil {
		panic(err)
	}
	totalDuration, err := probeDuration(a)
	if err != nil {
		panic(err)
	}

	err = ffmpeg.Input(inFileName).
		Output(outFileName, hevcArgs).
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
		progress := 0.00
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
			if cp != 0.00 && cp != progress {
				progress = cp
				Program.Send(progressMsg(progress))
				if (progress) == 1.00 {
					time.Sleep(time.Second / 2)
					Program.Send(progressMsg(1.5))
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
