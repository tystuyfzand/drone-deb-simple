package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	key := parseEnvOrFile("PLUGIN_KEY")

	urlStr := parseEnvOrFile("PLUGIN_URL")

	u, err := url.Parse(urlStr)

	if err != nil {
		fmt.Println("Invalid URL:", err)
		os.Exit(1)
	}

	query := u.Query()

	if arch := parseEnvOrFile("PLUGIN_ARCH"); arch != "" {
		query.Set("arch", arch)
	}

	if distro := parseEnvOrFile("PLUGIN_DISTRO"); distro != "" {
		query.Set("distro", distro)
	}

	if force := parseEnvOrFile("PLUGIN_FORCE"); force != "" && force != "false" {
		query.Set("force", "true")
	}

	u.RawQuery = query.Encode()

	var b bytes.Buffer

	m := multipart.NewWriter(&b)

	files := parseFiles()

	fmt.Println("Uploading files " + strings.Join(files, ", "))

	for i, file := range files {
		f, err := os.Open(file)

		if err != nil {
			fmt.Println("Unable to open file", file, ":", err)
			os.Exit(1)
		}

		fmt.Println("Attaching file " + file)

		fw, err := m.CreateFormFile("file_" + strconv.Itoa(i), f.Name())

		if err != nil {
			fmt.Println("Unable to create file part:", err)
			os.Exit(1)
			f.Close()
			continue
		}

		if _, err = io.Copy(fw, f); err != nil {
			fmt.Println("Unable to attach file:", err)
			os.Exit(1)
		}

		f.Close()
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), &b)

	if err != nil {
		fmt.Println("Unable to create http request:", err)
		os.Exit(1)
	}

	if key != "" {
		req.Header.Set("Authorization", "Token " + key)
	}

	req.Header.Set("Content-Type", m.FormDataContentType())

	c := &http.Client{Timeout: 15 * time.Second}

	res, err := c.Do(req)

	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		// Log error
		fmt.Println("Unable to upload packages:", err)
		os.Exit(1)
	}
}

func parseEnvOrFile(name string) string {
	fileEnv := os.Getenv(name + "_FILE")

	if fileEnv != "" {
		b, err := ioutil.ReadFile(fileEnv)

		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}

	return os.Getenv(name)
}

func parseFiles() []string {
	split := strings.Split(os.Getenv("PLUGIN_FILES"), ",")

	var files []string

	for _, p := range split {
		globed, err := filepath.Glob(p)

		if err != nil {
			panic("Unable to glob " + p + ": " + err.Error())
		}

		if globed != nil {
			files = append(files, globed...)
		}
	}

	return files
}