package main

import (
	"github.com/schollz/progressbar/v3"

	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var latestVersion string
var latestVersionURI string

func GetGoHTML() (string, error) {
	res, err := http.Get("https://go.dev/dl/#")
	if err != nil {
		return "", fmt.Errorf("error fetching Go download page: %w", err)
	}
	defer res.Body.Close()

	content, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	return string(content), nil
}

func ParseVersionRegex(URL string) string {
	x := regexp.MustCompile(`\d+\.\d+(\.\d+)?`)
	return x.FindString(URL)
}

func ParseDownloadURL() (string, error) {
	if latestVersionURI != "" {
		return "https://go.dev/" + latestVersionURI, nil
	}

	content, err := GetGoHTML()
	if err != nil {
		return "", errors.New("Couldn't Get URL")
	}

	re := regexp.MustCompile(`<a\s+class="download downloadBox"\s+href="(/dl/go\d+\.\d+\.\d+\.linux-amd64\.tar\.gz)">`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		latestVersionURI = matches[1]
		latestVersion = ParseVersionRegex(matches[1])
		return "https://go.dev/" + latestVersionURI, nil
	}

	return "", errors.New("Couldn't Parse URL")
}

func ParseLatestVersion() (string, error) {
	if latestVersion != "" {
		return latestVersion, nil
	}

	URL, err := ParseDownloadURL()
	if err != nil {
		return "", fmt.Errorf("failed to parse website")
	}

	latestVersion = ParseVersionRegex(URL)
	return latestVersion, nil
}

func InstallFromURL(URL, Path, Name string) error {
	Response, err := http.Get(URL)
	if err != nil {
		return fmt.Errorf("failed to initiate download: %w", err)
	}
	defer Response.Body.Close()

	Size, err := strconv.ParseInt(Response.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to determine file size: %w", err)
	}

	File := filepath.Join(Path, Name)
	Out, err := os.Create(File)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer Out.Close()

	Bar := progressbar.NewOptions(
		int(Size),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
	)

	_, err = io.Copy(io.MultiWriter(Out, Bar), Response.Body)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	Bar.Finish()

	return nil
}

func DLGoLatest() error {
	URL, err := ParseDownloadURL()
	if err != nil {
		return fmt.Errorf("error parsing download URL: %w", err)
	}

	if isVersionInstalled(latestVersion) {
		return fmt.Errorf("Latest Go version '%s' is already installed.", latestVersion)
	}

	err = InstallFromURL(URL, MangoPath+"/cache", latestVersion)
	if err != nil {
		return fmt.Errorf("error installing Go: %w", err)
	}

	err = ExtractVersion(latestVersion)
	if err != nil {
		return fmt.Errorf("error extracting Go: %w", err)
	}

	return nil
}

func DLGo(version string) error {
	err := InstallFromURL("https://go.dev/dl/go"+version+".linux-amd64.tar.gz", MangoPath+"/cache", version)
	if err != nil {
		return fmt.Errorf("error installing Go: %w", err)
	}

	err = ExtractVersion(version)
	if err != nil {
		return fmt.Errorf("error extracting Go: %w", err)
	}

	return nil
}

func isVersion(version string) bool {
	return regexp.MustCompile(`^\d+(\.\d+(\.\d+)?)?$`).MatchString(version)
}

func isValidVersion(version string) (bool, error) {
	content, err := GetGoHTML()
	if err != nil {
		return false, fmt.Errorf("error fetching Go HTML content: %w", err)
	}

	regex := regexp.MustCompile(`<div class="toggle" id="go` + version + `">`)
	match := regex.FindString(content)
	return match != "", nil
}
