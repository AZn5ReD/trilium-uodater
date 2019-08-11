package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mholt/archiver"
)

// https://api.github.com/repos/zadam/trilium/releases/latest

const repoUser string = "zadam"
const repoName string = "trilium"
const path string = "../"

func main() {
	log.Println("Starting Trilium updater")
	platform := getPlatform()

	filename, url, err := getDownloadUrl(platform)
	if err != nil {
		log.Fatal(err)
	}

	err = downloadFile(filename, url)
	if err != nil {
		log.Fatal(err)
	}

	err = unarchiveFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = cleanup(filename)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Update successful !")
}

func getPlatform() string {
	var os string
	switch runtime.GOOS {
	case "windows":
		log.Println("Windows OS detected")
		os = "windows"
	case "darwin":
		log.Println("macOS detected")
		os = "mac"
	default:
		log.Println("Linux detected")
		os = "linux"
	}
	return os
}

func getDownloadUrl(platform string) (string, string, error) {
	client := github.NewClient(nil)
	ctx := context.Background()

	log.Println("Fetching latest release asset list...")
	service, _, err := client.Repositories.GetLatestRelease(ctx, repoUser, repoName)
	if err != nil {
		return "", "", err
	}

	log.Println("Looking for relevant asset...")
	var asset github.ReleaseAsset
	for _, ass := range service.Assets {
		if strings.HasPrefix(*ass.Name, repoName+"-"+platform) && !(strings.Contains(*ass.Name, "server")) {
			asset = ass
			break
		}
	}
	if asset.Name == nil {
		return "", "", errors.New("Couldn't find relevant asset")
	}

	log.Println("Getting download url")
	_, url, err := client.Repositories.DownloadReleaseAsset(ctx, repoUser, repoName, asset.GetID())
	if err != nil {
		return "", "", err
	}
	return *asset.Name, url, nil
}

func downloadFile(filename string, url string) error {
	filepath := path + filename

	log.Println("Downloading " + filename + "...")
	log.Println("GET Request " + url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Println("Create file " + filepath)
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	log.Println("Writing file...")
	_, err = io.Copy(out, resp.Body)
	if err == nil {
		log.Println("Download successful !")
	}
	return err
}

func unarchiveFile(filename string) error {
	log.Println("Detect extension...")
	ar, err := archiver.ByExtension(filename)
	if err != nil {
		return err
	}
	log.Println(ar)

	log.Println("Setting overwrite")
	switch ar.(type) {
	case *archiver.TarXz:
		ar = &archiver.TarXz{Tar: &archiver.Tar{OverwriteExisting: true}}
	case *archiver.Zip:
		ar = &archiver.Zip{OverwriteExisting: true}
	default:
		return errors.New("Can't detect extension")
	}

	log.Println("Extracting archive...")
	err = ar.(archiver.Unarchiver).Unarchive(path+filename, path)
	if err == nil {
		log.Println("Done !")
	}
	return err
}

func cleanup(filename string) error {
	filepath := path + filename

	log.Println("Deleting archive")
	err := os.Remove(filepath)
	return err
}
