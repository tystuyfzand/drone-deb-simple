package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	key := parseEnvOrFile("PLUGIN_KEY")

	urlStr := parseEnvOrFile("PLUGIN_URL")

	files := parseFiles()

	err := Upload(urlStr, key, files)

	if err != nil {
		fmt.Println("Unable to upload:", err)
		os.Exit(1)
	}
}

func Upload(urlStr, key string, files []string) error {
	u, err := url.Parse(urlStr)

	if err != nil {
		return err
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

	r, w := io.Pipe()

	m := multipart.NewWriter(w)

	fmt.Println("Uploading files " + strings.Join(files, ", "))

	var size, partSize int64

	// Cheap trick to calculate upload size based on files we're uploading so we can stream it all

	boundaryLen := int64(len(m.Boundary())) + 2

	for i, file := range files {
		stat, err := os.Stat(file)

		if os.IsNotExist(err) {
			return err
		}

		// Size = file size + boundary length + \r\n
		partSize = stat.Size() + boundaryLen + 1

		if i > 0 {
			partSize += 2
		}

		// Content Type + Content Disposition
		partSize += 38 + 52

		// Names of fields
		partSize += int64(len(escapeQuotes(path.Base(file))))
		partSize += int64(len("form_" + strconv.Itoa(i)))

		// Newline spacing
		partSize += 8

		size += partSize
	}

	size += boundaryLen + 2

	go func() {
		defer m.Close()

		for i, file := range files {
			f, err := os.Open(file)

			if err != nil {
				fmt.Println("Unable to open file", file, ":", err)
				os.Exit(1)
			}

			fmt.Println("Attaching file " + path.Base(file))

			fw, err := m.CreateFormFile("file_"+strconv.Itoa(i), path.Base(f.Name()))

			if err != nil {
				fmt.Println("Unable to create file part:", err)
				os.Exit(1)
			}

			if _, err = io.Copy(fw, f); err != nil {
				fmt.Println("Unable to attach file:", err)
				os.Exit(1)
			}

			f.Close()
		}
	}()

	req, err := http.NewRequest(http.MethodPost, u.String(), r)

	if err != nil {
		return err
	}

	req.ContentLength = size

	req.Header.Set("Content-Length", strconv.FormatInt(size, 10))
	req.Header.Set("User-Agent", "DroneDebSimple (v0.1, "+runtime.Version()+")")
	req.Header.Set("Content-Type", m.FormDataContentType())

	if key != "" {
		req.Header.Set("Authorization", "Token "+key)
	}

	c := &http.Client{Timeout: 120 * time.Second}

	res, err := c.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	bb, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d, response %s", res.StatusCode, string(bb))
	}

	return nil
}

// Load a variable from either name or name + "_FILE"
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

// Construct a list of files based on a given env variable, as glob input
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

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
