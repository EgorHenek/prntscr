package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/otiai10/gosseract/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const codeChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func main() {
	config()

	code := viper.GetString("StartCode")

	client := gosseract.NewClient()
	defer client.Close()

	for {
		html, err := getHTML(code)
		if err != nil {
			log.Fatalln(err, code)
		}

		imgURL, err := getImgURL(html)
		if err != nil {
			log.Fatalln(err, code)
		}

		if strings.HasPrefix(imgURL, "//") {
			imgURL = "https:" + imgURL
		}

		path, err := downloadImg(code, imgURL)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				code, err = increaseCode(code)
				if err != nil {
					log.Fatalln(err, code)
				}
				continue
			}
			log.Fatalln(err, code)
		}

		imageText, err := scanImg(*client, path)
		if err != nil {
			log.Fatalln(err, code)
		}
		word := findKeywords(imageText)
		if word != "" {
			log.Printf("%v: %v", imgURL, word)
		}

		code, err = increaseCode(code)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func getHTML(code string) ([]byte, error) {
	var html []byte

	client := &http.Client{}
	req, err := http.NewRequest("GET", viper.GetString("BaseURL")+code, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting html with code: %v: %v", code, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting html with code: %v: %v", code, response.Status)
	}
	defer response.Body.Close()
	html, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading html with code: %v: %v", code, err)
	}
	return html, nil
}

func getImgURL(html []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("error creating document: %v", err)
	}
	imgURL, exists := doc.Find("img.screenshot-image").Attr("src")
	if !exists {
		return "", fmt.Errorf("error getting image url")
	}
	return imgURL, nil
}

func downloadImg(code, imageUri string) (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: viper.GetString("proxy.scheme"),
				Host:   viper.GetString("proxy.host"),
				User:   url.UserPassword(viper.GetString("proxy.user"), viper.GetString("proxy.pass")),
			}),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest("GET", imageUri, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error getting image: %v, URL: %v", err, imageUri)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting image: %v, URL: %v", response.StatusCode, imageUri)
	}
	defer response.Body.Close()

	path := viper.GetString("ImagesDir") + "/" + code + ".png"
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("error copying image to file: %v", err)
	}
	return path, nil
}

func scanImg(client gosseract.Client, path string) (string, error) {
	err := client.SetImage(path)
	if err != nil {
		return "", fmt.Errorf("error setting image: %v", err)
	}
	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("error getting text: %v", err)
	}
	return text, nil
}

func config() {
	pflag.String("ImagesDir", "images", "Directory to store images")
	pflag.StringP("StartCode", "s", "sjgmm9", "Starting code")
	pflag.StringP("ProxyScheme", "p", "http", "Proxy scheme")
	pflag.StringP("ProxyHost", "h", "", "Proxy host:port")
	pflag.StringP("ProxyUser", "u", "", "Proxy user")
	pflag.StringP("ProxyPass", "w", "", "Proxy password")
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	viper.SetDefault("ImagesDir", "images")
	viper.SetDefault("BaseURL", "https://prnt.sc/")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()
}

func findKeywords(text string) string {
	keywords := viper.GetStringSlice("Keywords")
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return fmt.Sprintf("Found %v", keyword)
		}
	}
	return ""
}

func increaseCode(code string) (string, error) {
	chars := []byte(codeChars)
	codeBytes := []byte(code)
	for i := len(codeBytes) - 1; i >= 0; i-- {
		if codeBytes[i] == chars[len(chars)-1] {
			if i == 0 {
				return "", fmt.Errorf("code is too long")
			}
			codeBytes[i] = chars[0]
			continue
		}
		codeBytes[i] = chars[strings.IndexByte(codeChars, codeBytes[i])+1]
		break
	}
	return string(codeBytes), nil
}
