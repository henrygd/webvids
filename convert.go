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

type progressMsg struct {
	percent    float64
	conversion string
}

type conversionDone string

type probeData struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

var CurentConversion = ""
var inputProbeData = probeData{}
var inputHeight = 0
var inputWidth = 0

var SpeedPresets = []struct {
	libx265   string
	libsvtav1 string
}{
	{
		libx265:   "slow",
		libsvtav1: "4",
	},
	{
		libx265:   "slow",
		libsvtav1: "5",
	},
	{
		libx265:   "medium",
		libsvtav1: "7",
	},
	{
		libx265:   "fast",
		libsvtav1: "9",
	},
	{
		libx265:   "veryfast",
		libsvtav1: "10",
	},
	{
		libx265:   "superfast",
		libsvtav1: "12",
	},
}

func Convert(infile string, outfile string, codec string) {
	// create optimized directory if it doesn't exist
	if _, err := os.Stat("./optimized"); os.IsNotExist(err) {
		os.Mkdir("./optimized", 0755)
	}

	// probe input file
	if len(inputProbeData.Streams) == 0 {
		a, err := ffmpeg.Probe(infile)
		CheckError(err)
		err = json.Unmarshal([]byte(a), &inputProbeData)
		CheckError(err)
	}

	// find width / height of video stream
	if inputWidth == 0 {
		for _, stream := range inputProbeData.Streams {
			if stream.Width != 0 && stream.Height != 0 {
				inputWidth = stream.Width
				inputHeight = stream.Height
				break
			}
		}
	}

	// create ffmpeg args
	ffmpegArgs := ffmpeg.KwArgs{}
	ffmpegArgs["c:v"] = codec

	// set resolution
	if inputWidth > inputHeight && inputHeight > 1080 {
		ffmpegArgs["vf"] = "scale=-2:1080"
	} else if inputHeight >= inputWidth && inputWidth > 1080 {
		ffmpegArgs["vf"] = "scale=1080:-2"
	}

	if Preview {
		ffmpegArgs["ss"] = "00:00:00"
		ffmpegArgs["t"] = "00:00:03"
	}

	if StripAudio {
		ffmpegArgs["an"] = ""
	}
	// else {
	// set audio options
	// ffmpegArgs["c:a"] = "libopus"
	// ffmpegArgs["b:a"] = "128k"
	// ffmpegArgs["ac"] = "2"
	// }

	// x265 specific options
	if codec == "libx265" {
		ffmpegArgs["crf"] = Crf
		ffmpegArgs["preset"] = SpeedPresets[Speed].libx265
		ffmpegArgs["movflags"] = "faststart"
		ffmpegArgs["tag:v"] = "hvc1"
		// ffmpegArgs["profile:v"] = "main"
		// ffmpegArgs["pix_fmt"] = "yuv420p"

	}
	// av1 specific options
	if codec == "libsvtav1" {
		ffmpegArgs["preset"] = SpeedPresets[Speed].libsvtav1
		ffmpegArgs["svtav1-params"] = "tune=0:enable-qm=1:qm-min=0:enable-tf=0"
		// ffmpegArgs["pix_fmt"] = "yuv420p10le"

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
	var err error

	// get duration of video (3 seconds if preview mode)
	totalDuration := 3.00
	if !Preview {
		totalDuration, err = probeDuration(inputProbeData)
		CheckError(err)
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

	// wait a sec to finish progress bar animation
	time.Sleep(time.Millisecond * 500)

	Program.Send(conversionDone(CurentConversion))
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
			if cp > 0.00 && cp <= 1.00 {
				Program.Send(progressMsg(progressMsg{
					percent:    cp,
					conversion: CurentConversion,
				}))
				if cp == 1.00 {
					l.Close()
					break
				}
			}
		}
	}()

	return sockFileName
}

func probeDuration(data probeData) (float64, error) {
	f, err := strconv.ParseFloat(data.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}
