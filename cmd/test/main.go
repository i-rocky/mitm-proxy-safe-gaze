package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/AdguardTeam/golibs/log"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var base64ImageRegex = regexp.MustCompile(`data:image/(png|jpeg|jpg|bmp|webp);base64,(.*?)[^\w=/+\\]`)

type Base64Image struct {
	Start       int
	End         int
	Original    []byte
	Replacement []byte
	Id          int
	Ext         string
	MD5         string
	Ref         string
}

func getMD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func replaceBase64Images(content string) string {
	if len(content) == 0 {
		return content
	}

	images := base64ImageRegex.FindAllStringSubmatchIndex(content, -1)
	if len(images) == 0 {
		return content
	}

	pickedImages := make([]Base64Image, 0)
	contentBytes := []byte(content)

	for _, image := range images {
		firstIndex := image[0]
		lastIndex := image[1] - 1
		if lastIndex-firstIndex >= 1000 {
			pickedImages = append(pickedImages, Base64Image{
				Start:       firstIndex,
				End:         lastIndex,
				Replacement: contentBytes[firstIndex:lastIndex],
				Ext:         string(contentBytes[image[2]:image[3]]),
				MD5:         getMD5(string(contentBytes[firstIndex:lastIndex])),
			})
		}
	}

	for i, image := range pickedImages {
		if image.Ref != "" {
			continue
		}
		for j := i + 1; j < len(pickedImages); j++ {
			if image.MD5 == pickedImages[j].MD5 {
				pickedImages[j].Ref = image.Ref
			}
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(pickedImages))
	for i, image := range pickedImages {
		go func(i int, image Base64Image) {
			defer wg.Done()
			image.Id = i
			image.Original = contentBytes[image.Start:image.End]
			withoutPrefix := image.Original[strings.Index(string(image.Original), ",")+1:]
			req, err := http.NewRequest("POST", "https://safe-gaze.clapbox.net/q", bytes.NewReader(withoutPrefix))
			if err != nil {
				log.Printf("failed to build request: %v", err)
				return
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("failed to make request: %v", err)
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Printf("failed to read response: %v", err)
				return
			}

			pickedImages[i].Replacement = make([]byte, base64.StdEncoding.EncodedLen(len(body)))
			base64.StdEncoding.Encode(pickedImages[i].Replacement, body)
			withoutPrefix = image.Replacement[strings.Index(string(image.Replacement), ",")+1:]
			data := pickedImages[i].Replacement
			prefix := []byte("data:image/jpeg;base64,")
			pickedImages[i].Replacement = make([]byte, len(prefix)+base64.StdEncoding.EncodedLen(len(data)))
			pickedImages[i].Replacement = append(prefix, data...)
		}(i, image)
	}

	wg.Wait()
	imageMap := make(map[string]Base64Image)
	for _, image := range pickedImages {
		imageMap[image.MD5] = image
	}

	for i, image := range pickedImages {
		if image.Ref == "" {
			continue
		}

		image.Replacement = imageMap[image.Ref].Replacement
		pickedImages[i] = image
	}

	nContent := make([]byte, 0)
	nContent = append(nContent, contentBytes[:pickedImages[0].Start]...)
	nContent = append(nContent, pickedImages[0].Replacement...)

	for i := 1; i < len(pickedImages); i++ {
		nContent = append(nContent, contentBytes[pickedImages[i-1].End:pickedImages[i].Start]...)
		nContent = append(nContent, pickedImages[i].Replacement...)
	}

	nContent = append(nContent, contentBytes[pickedImages[len(pickedImages)-1].End:]...)

	return string(nContent)
}

func main() {
	_ = os.RemoveAll("cmd/test/debug")
	_ = os.MkdirAll("cmd/test/debug", 0755)

	file, err := os.Open("cmd/test/resp.html")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	timeStart := time.Now()

	nContent := replaceBase64Images(string(content))

	timeEnd := time.Now()

	log.Printf("Time taken: %s", timeEnd.Sub(timeStart))

	nFile, err := os.Create("cmd/test/resp-new.html")
	if err != nil {
		panic(err)
	}
	defer nFile.Close()

	_, err = nFile.Write([]byte(nContent))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Done")
}
