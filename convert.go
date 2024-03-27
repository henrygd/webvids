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

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var CurentConversion = ""

type progressMsg struct {
	percent    float64
	conversion string
}

func Convert(infile string, outfile string, codec string) {
	// create optimized directory if it doesn't exist
	if _, err := os.Stat("./optimized"); os.IsNotExist(err) {
		os.Mkdir("./optimized", 0755)
	}

	ffmpegArgs := ffmpeg.KwArgs{
		"vf": "scale=-1:1080",
	}
	ffmpegArgs["c:v"] = codec
	if StripAudio {
		ffmpegArgs["an"] = ""
	}
	if Preview {
		ffmpegArgs["ss"] = "00:00:00"
		ffmpegArgs["t"] = "00:00:03"
	}
	// x265 specific options
	if codec == "libx265" {
		ffmpegArgs["crf"] = Crf
		ffmpegArgs["movflags"] = "faststart"
		ffmpegArgs["preset"] = "slow"
		ffmpegArgs["tag:v"] = "hvc1"
		// ffmpegArgs["profile:v"] = "main"
		// ffmpegArgs["pix_fmt"] = "yuv420p"

	}
	// av1 specific options
	if codec == "libsvtav1" {
		ffmpegArgs["preset"] = "6"
		// add 7 to base crf (28 -> 35)
		crf, err := strconv.Atoi(Crf)
		if err != nil {
			fmt.Println("Failed to convert string to integer:", err)
			os.Exit(1)
		}
		crf += 7
		ffmpegArgs["crf"] = strconv.Itoa(crf)
	}
	CurentConversion = codec
	convertWithProgress(infile, outfile, ffmpegArgs)
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

	Cmd = ffmpeg.Input(inFileName).
		Output(outFileName, ffmpegArgs).
		GlobalArgs("-progress", "unix://"+TempSock(totalDuration)).
		OverWriteOutput().
		Silent(true).
		Compile()

	err = Cmd.Run()
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
