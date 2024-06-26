package main

import (
	"github.com/spf13/cobra"

	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const MangoVersion = "0.1.0"

var Install = &cobra.Command{
	Use:       "install <version>",
	Short:     "Download Go versions",
	Aliases:   []string{"download"},
	ValidArgs: []string{"latest"},
	Args:      cobra.ExactArgs(1),
	Run:       Install_CLI,
}

func init() { // why are flags so ugly should be applicated in the cobra.command?
	Install.Flags().BoolP("use", "u", false, "Set Go environment after installation")
}

func CLI() {
	root := &cobra.Command{
		Use:   "mango",
		Short: "Go Version Manager",
		Long:  "Mango is a command-line tool that simplifies the installation and management of multiple Go versions.",
	}

	root.DisableFlagsInUseLine = false
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		Install,
		&cobra.Command{
			Use:               "uninstall <version>",
			Short:             "Remove Go versions",
			Aliases:           []string{"remove"},
			Args:              cobra.ExactArgs(1),
			Run:               Uninstall_CLI,
			ValidArgsFunction: Version_ARG,
		},
		&cobra.Command{
			Use:               "use <version>",
			Short:             "Select Go version",
			Aliases:           []string{"set"},
			Args:              cobra.ExactArgs(1),
			Run:               Use_CLI,
			ValidArgsFunction: Version_ARG,
		},
		&cobra.Command{
			Use:     "list",
			Short:   "Show Go versions",
			Aliases: []string{"show"},
			Args:    cobra.ExactArgs(0),
			Run:     List_CLI,
		},
		&cobra.Command{
			Use:   "version",
			Short: "Build Information",
			Args:  cobra.ExactArgs(0),
			Run:   Version_CLI,
		},
		&cobra.Command{
			Use:    "completion [bash|zsh|fish]",
			Short:  "Generate completion script",
			Args:   cobra.ExactArgs(1),
			Hidden: true,
			Run:    Completion_CLI,
		},
	)

	if err := root.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func Completion_CLI(cmd *cobra.Command, args []string) {
	Root := cmd.Root()
	switch args[0] {
	case "bash":
		Root.GenBashCompletion(os.Stdout)
	case "zsh":
		Root.GenZshCompletion(os.Stdout)
	case "fish":
		Root.GenFishCompletion(os.Stdout, true)
	default:
		fmt.Println("Unsupported shell.")
		os.Exit(1)
	}
}

func Version_ARG(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	Versions, err := GetVersions()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var Valid []string
	for _, Version := range Versions {
		if strings.HasPrefix(Version, toComplete) {
			Valid = append(Valid, Version)
		}
	}

	if strings.Split(cmd.CommandPath(), " ")[1] == "use" {
		Valid = append(Valid, "latest")
	}

	return Valid, cobra.ShellCompDirectiveNoFileComp
}

func Version_CLI(cmd *cobra.Command, args []string) {
	fmt.Printf("Mango: %s\n", MangoVersion)

	Version, err := GetVersion()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Use 'mango install <version>' to add Go versions.")
		} else {
			fmt.Println("Error retrieving Go version:", err)
		}
		return
	}

	fmt.Printf("Go: %s\n", Version)
}

func Use_CLI(cmd *cobra.Command, args []string) {
	Version := args[0]

	if Version == "latest" {
		LatestVersion, err := GetLatestVersion()
		if err != nil {
			fmt.Println(err)
			return
		}
		Version = LatestVersion
	}

	SymlinkVersion, err := GetVersion()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Println("Error unable to fetch the Go version")
		return
	}

	if Version == SymlinkVersion {
		fmt.Printf("Go environment is on version %s\n", Version)
		return
	}

	if isVersionInstalled(Version) {
		err := SwitchVersion(Version)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Go environment is now using version %s\n", Version)
	} else {
		if isVersion(Version) {
			Valid, err := isValidVersion(Version)
			if err != nil {
				fmt.Println("Error checking version availability, try again or check your internet connection.")
				return
			}

			if Valid {
				fmt.Printf("Go version %s couldn't be found, use 'mango install %s' to download it.\n", Version, Version)
			} else {
				fmt.Println("Unsupported Go version, refer to the official download page for available options.")
			}
		} else {
			fmt.Println("Invalid Go version specified.")
		}
	}
}

func List_CLI(cmd *cobra.Command, args []string) {
	entries, err := os.ReadDir(filepath.Join(MangoPath, "version"))
	if err != nil {
		return
	}

	if len(entries) > 0 {
		var versions []string
		for _, entry := range entries {
			versions = append(versions, entry.Name())
		}

		sort.Slice(versions, func(i, j int) bool {
			return versions[i] > versions[j]
		})

		fmt.Println("Installed Go Versions:")
		for _, version := range versions {
			fmt.Println(version)
		}
	} else {
		fmt.Println("Use 'mango install <version>' to add Go versions.")
	}
}

func Uninstall_CLI(cmd *cobra.Command, args []string) {
	Version := args[0]

	if isVersion(Version) {
		if !isVersionInstalled(Version) {
			fmt.Printf("Go version %s is not installed.\n", Version)
			return
		}

		err := RemoveVersion(Version)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Go version %s was uninstalled.\n", Version)

	} else {
		fmt.Println("Invalid Go version: use 'mango list' to view installed versions to uninstall.")
	}

	SymlinkVersion, err := GetVersion()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = AutoVersionSwitch()
			if err != nil {
				fmt.Println("Error auto-switching versions after uninstall:", err)
			}
		} else {
			fmt.Println("Error unable to fetch the Go version")
		}
		return
	}

	if SymlinkVersion == Version {
		err = CleanBinSymlink()
		if err != nil {
			fmt.Println("Error cleaning symlinks:", err)
		}
	}

	err = AutoVersionSwitch()
	if err != nil {
		fmt.Println("Error auto-switching versions after download:", err)
	}
}

func Install_CLI(cmd *cobra.Command, args []string) {
	Version := args[0]

	if Version == "latest" {
		err := DLGoLatest()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Go version %s is now installed.\n", latestVersion)

		err = AutoVersionSwitch()
		if err != nil {
			fmt.Println("Error auto-switching versions after download:", err)
		}
	} else {
		if isVersion(Version) {
			if isVersionInstalled(Version) {
				fmt.Printf("Go version %s is already installed.\n", Version)
				return
			}

			Valid, err := isValidVersion(Version)
			if err != nil {
				fmt.Println("Error checking version availability:", err)
				return
			}

			if !Valid {
				fmt.Println("Invalid Go version:", Version)
				return
			}

			err = DLGo(Version)
			if err != nil {
				fmt.Println("Error downloading Go version:", err)
			}

			fmt.Printf("Go version %s is now installed.\n", Version)
		} else {
			fmt.Println("Invalid Go version: use a specific version or 'latest' for the most recent version.")
		}
	}

	Use, err := cmd.Flags().GetBool("use")
	if err != nil {
		fmt.Println(err)
		return
	}

	if Use == true {
		if Version == "latest" {
			Version, err = ParseLatestVersion()
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		err = SwitchVersion(Version)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Go environment is now using version %s\n", Version)
		return
	}

	err = AutoVersionSwitch()
	if err != nil {
		fmt.Println("Error auto-switching versions after download:", err)
	}
}
