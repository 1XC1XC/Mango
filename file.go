package main

import (
	"github.com/schollz/progressbar/v3"

	"archive/tar"
	"compress/gzip"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var MangoPath string

func init() {
	Path, err := GetMangoPath()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	CleanMangoCache(Path)
	MangoPath = Path
}

func GetMangoPath() (string, error) {
	Home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Cannot determine your home directory. 'mango' requires access to this location.")
	}
	Home = filepath.Join(Home, ".mango")
	if _, err := os.Stat(Home); os.IsNotExist(err) {
		return "", fmt.Errorf("Missing '.mango' directory. Please ensure 'mango' was installed correctly.")
	}
	return Home, nil
}

func CleanMangoCache(Home string) {
	CachePath := filepath.Join(Home, "cache")
	Files, err := os.ReadDir(CachePath)
	if err != nil {
		fmt.Printf("Error reading cache directory: %v\n", err)
		return
	}

	for _, File := range Files {
		Path := filepath.Join(CachePath, File.Name())
		if File.IsDir() {
			err = os.RemoveAll(Path)
		} else {
			err = os.Remove(Path)
		}
		if err != nil {
			fmt.Printf("Error removing %s: %v\n", Path, err)
		}
	}
}

func CleanBinSymlink() error {
	Bin := filepath.Join(MangoPath, "bin")
	Files, err := os.ReadDir(Bin)
	if err != nil {
		return err
	}

	for _, File := range Files {
		if File.Name() == "mango" || File.IsDir() {
			continue
		}
		Symlink := filepath.Join(Bin, File.Name())
		if isExecutable(Symlink) {
			err = os.Remove(Symlink)
			if err != nil {
				return fmt.Errorf("Error removing symlink: %v\n", err)
			}
		}
	}

	return nil
}

func RemoveVersion(Version string) error {
	VD := filepath.Join(MangoPath, "version", Version)

	if _, err := os.Stat(VD); os.IsNotExist(err) {
		return fmt.Errorf("Go version %s was not found", Version)
	}

	err := os.RemoveAll(VD)
	if err != nil {
		return fmt.Errorf("Error removing version '%s': "+err.Error()+"\n", Version)
	}

	return nil
}

func SwitchVersion(version string) error {
	if !isVersionInstalled(version) {
		return fmt.Errorf("version %s is not installed", version)
	}

	Link := filepath.Join(MangoPath, "bin")
	Symlink := filepath.Join(MangoPath, "version", version, "bin")

	Files, err := os.ReadDir(Symlink)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", Symlink, err)
	}

	for _, File := range Files {
		Name := File.Name()
		Source := filepath.Join(Symlink, Name)

		if !isExecutable(Source) {
			continue
		}

		Target := filepath.Join(Link, Name)

		err = CreateSymlink(Source, Target)
		if err != nil {
			return fmt.Errorf("error creating symlink for %s: %w", Name, err)
		}
	}

	return nil
}

func AutoVersionSwitch() error {
	Versions, err := os.ReadDir(filepath.Join(MangoPath, "version"))
	if err != nil {
		return err
	}

	if len(Versions) == 1 {
		Version := Versions[0].Name()
		err = SwitchVersion(Version)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateSymlink(source, target string) error {
	if _, err := os.Lstat(target); err == nil {
		err = os.Remove(target)
		if err != nil {
			return fmt.Errorf("error removing existing symlink at %s: %w", target, err)
		}
	}

	err := os.Symlink(source, target)
	if err != nil {
		return fmt.Errorf("error creating symlink from %s to %s: %w", source, target, err)
	}

	err = os.Chmod(target, 0755)
	if err != nil {
		return fmt.Errorf("error setting symlink permissions: %w", err)
	}

	return nil
}

func isExecutable(Path string) bool {
	File, err := os.Open(Path)
	if err != nil {
		return false
	}
	defer File.Close()

	ELF, err := elf.NewFile(File)
	if err != nil {
		return false
	}

	Header := ELF.FileHeader
	if Header.Type != elf.ET_EXEC && Header.Type != elf.ET_DYN {
		return false
	}

	return true
}

func isVersionInstalled(version string) bool {
	Target := filepath.Join(MangoPath, "version", version)
	_, err := os.Stat(Target)
	return err == nil
}

func GetVersion() (string, error) {
	GoPath := filepath.Join(MangoPath, "bin", "go")
	if _, err := os.Stat(GoPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", os.ErrNotExist
		} else {
			return "", fmt.Errorf("error checking for 'go' executable: %w", err)
		}
	}

	Symlink, err := os.Readlink(GoPath)
	if err != nil {
		return "", fmt.Errorf("error reading 'go' symlink: %w", err)
	}

	Target := filepath.Dir(Symlink)
	VersionDir := filepath.Dir(Target)
	Version := filepath.Base(VersionDir)

	return Version, nil
}

func ExtractVersion(Version string) error {
	Cache := filepath.Join(MangoPath, "cache", Version)
	Target := filepath.Join(MangoPath, "version", Version)
	return ExtractTarGz(Cache, Target, Version)
}

func ExtractTarGz(ArchivePath string, TargetDir string, ExtractDirName string) error {
	Archive, err := os.Open(ArchivePath)
	if err != nil {
		return fmt.Errorf("ExtractTarGz: error opening archive: %w", err)
	}
	defer Archive.Close()

	GzipReader, err := gzip.NewReader(Archive)
	if err != nil {
		return fmt.Errorf("ExtractTarGz: error creating Gzip reader: %w", err)
	}
	defer GzipReader.Close()

	TarReader := tar.NewReader(GzipReader)

	var Prefix string

	Count, err := GetEntryCount(ArchivePath)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	PB := progressbar.NewOptions(
		Count,
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowElapsedTimeOnFinish(), // probably remove this line
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
	)

	for {
		Header, err := TarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractTarGz: error reading tar entry: %w", err)
		}

		if Prefix == "" && Header.Typeflag == tar.TypeDir {
			Prefix = Header.Name
		}

		FullPath := filepath.Join(TargetDir, strings.TrimPrefix(Header.Name, Prefix))

		switch Header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(FullPath, 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: error creating directory: %w", err)
			}
		case tar.TypeReg:
			OutFile, err := os.Create(FullPath)
			if err != nil {
				return fmt.Errorf("ExtractTarGz: error creating file: %w", err)
			}
			defer OutFile.Close()

			_, err = io.Copy(OutFile, TarReader)
			if err != nil && err != io.EOF {
				return fmt.Errorf("ExtractTarGz: error copying file content: %w", err)
			}

			PB.Add(1)
		}
	}

	PB.Finish()

	return nil
}

func GetEntryCount(Archive string) (int, error) {
	File, err := os.Open(Archive)
	if err != nil {
		return 0, err
	}
	defer File.Close()

	GZR, err := gzip.NewReader(File)
	if err != nil {
		return 0, err
	}
	defer GZR.Close()

	TR := tar.NewReader(GZR)

	Count := 0
	for {
		_, err := TR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("GetEntryCount: error reading tar entry: %w", err)
		}
		Count++
	}

	return Count, nil
}
