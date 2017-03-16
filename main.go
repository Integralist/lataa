package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	fastly "github.com/sethvargo/go-fastly"
)

// Fastly API doesn't return sorted data
type version struct {
	Number  int
	Version *fastly.Version
}

type wrappedVersions []version

// Satisfy the Sort interface
func (v wrappedVersions) Len() int      { return len(v) }
func (v wrappedVersions) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v wrappedVersions) Less(i, j int) bool {
	return v[i].Number < v[j].Number
}

type vclResponse struct {
	Path    string
	Name    string
	Content string
	Error   bool
}

// Globals needed for sharing between functions
var fastlyServiceID string
var latestVersion string
var selectedVersion string

// List of VCL files to process
var vclFiles []string

// Regex used to define user specific filtering
var dirSkipRegex *regexp.Regexp
var dirMatchRegex *regexp.Regexp

// WaitGroup and Channel for storing responses from API
var wg sync.WaitGroup
var ch chan vclResponse

// Useful colour settings for printing messages
var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()

// Version is the application version
const Version = "1.1.0"

func main() {
	help := flag.Bool("help", false, "show available flags")
	appVersion := flag.Bool("version", false, "show application version")
	useLatestVersion := flag.Bool("use-latest-version", false, "use latest Fastly service version to upload to (presumes not activated)")
	getLatestVersion := flag.Bool("get-latest-version", false, "get latest Fastly service version and its active status")
	cloneVersion := flag.String("clone-version", "", "specify Fastly service 'version' to clone from before uploading to")
	uploadVersion := flag.String("upload-version", "", "specify non-active Fastly service 'version' to upload to")
	activateVersion := flag.String("activate-version", "", "specify Fastly service 'version' to activate")
	statusVersion := flag.String("get-version-status", "", "retrieve status for the specified Fastly service 'version'")
	service := flag.String("service", os.Getenv("FASTLY_SERVICE_ID"), "your service id (fallback: FASTLY_SERVICE_ID)")
	token := flag.String("token", os.Getenv("FASTLY_API_TOKEN"), "your fastly api token (fallback: FASTLY_API_TOKEN)")
	dir := flag.String("dir", os.Getenv("VCL_DIRECTORY"), "vcl directory to upload files from")
	skip := flag.String("skip", "^____", "regex for skipping vcl directories (will also try: VCL_SKIP_DIRECTORY)")
	match := flag.String("match", "", "regex for matching vcl directories (will also try: VCL_MATCH_DIRECTORY)")
	flag.Parse()

	if *help == true {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *appVersion == true {
		fmt.Println(Version)
		os.Exit(1)
	}

	envSkipDir := os.Getenv("VCL_SKIP_DIRECTORY")
	envMatchDir := os.Getenv("VCL_MATCH_DIRECTORY")

	if envSkipDir != "" {
		*skip = envSkipDir
	}

	if envMatchDir != "" {
		*match = envMatchDir
	}

	dirSkipRegex, _ = regexp.Compile(*skip)
	dirMatchRegex, _ = regexp.Compile(*match)

	client, err := fastly.NewClient(*token)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fastlyServiceID = *service

	if *getLatestVersion {
		latestVersion, status, err := getLatestServiceVersion(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("\nLatest service version: %s (%s)\n\n", latestVersion, status)
		return
	}

	// Activate Version
	if *activateVersion != "" {
		_, err := client.ActivateVersion(&fastly.ActivateVersionInput{
			Service: fastlyServiceID,
			Version: *activateVersion,
		})
		if err != nil {
			fmt.Printf("\nThere was a problem activating version %s\n\n%s", yellow(*activateVersion), red(err))
			os.Exit(1)
		}
		fmt.Printf("\nService '%s' now has version '%s' activated\n\n", yellow(fastlyServiceID), green(*activateVersion))
		return
	}

	// Version Status Check
	if *statusVersion != "" {
		status, err := getStatusVersion(*statusVersion, client)
		if err != nil {
			fmt.Printf("\nThere was a problem getting the status for version %s\n\n%s\n\n", yellow(*statusVersion), red(err))
			os.Exit(1)
		}
		fmt.Printf("\nService '%s' version '%s' is '%s'\n\n", yellow(fastlyServiceID), yellow(*statusVersion), status)
		return
	}

	// Incorrect flags provided check
	if *cloneVersion != "" && *uploadVersion != "" {
		fmt.Println("Please do not provide both -clone-version and -upload-version flags")
		os.Exit(1)
	}

	// Check if we should...
	// 		A. clone the specified version before uploading files: `-clone-version`
	// 		B. upload files to the specified version: `-upload-version`
	// 		C. upload files to the latest version: `-use-latest-version`
	// 		D. clone the latest version if it's already activated

	// Clone from specified version and upload to that
	if *cloneVersion != "" {
		clonedVersion, err := cloneFromVersion(*cloneVersion, client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created new version %s from existing version %s\n\n", clonedVersion.Number, *cloneVersion)
		selectedVersion = *cloneVersion
	} else if *uploadVersion != "" {
		// Upload to the specified version (it can't be activated)
		getVersion, err := client.GetVersion(&fastly.GetVersionInput{
			Service: fastlyServiceID,
			Version: *uploadVersion,
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if getVersion.Active {
			fmt.Println("Sorry, the specified version is already activated")
			os.Exit(1)
		}
		selectedVersion = *uploadVersion
	} else {
		latestVersion, err := getLatestVCLVersion(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		selectedVersion = latestVersion

		// Upload to the latest version (it can't be activated)
		if *useLatestVersion {
			getVersion, err := client.GetVersion(&fastly.GetVersionInput{
				Service: fastlyServiceID,
				Version: latestVersion,
			})
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if getVersion.Active {
				fmt.Println("Sorry, the latest version is already activated")
				os.Exit(1)
			}
			selectedVersion = latestVersion
		} else {
			// Otherwise clone the latest version and upload to that
			clonedVersion, err := cloneFromVersion(latestVersion, client)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Successfully created new version %s from latest version %s\n\n", clonedVersion.Number, latestVersion)
			selectedVersion = clonedVersion.Number
		}
	}

	// Recursively walk the specified directory: `-dir` or `VCL_DIRECTORY`
	walkError := filepath.Walk(*dir, aggregate)
	if walkError != nil {
		fmt.Printf("filepath.Walk() returned an error: %v\n", walkError)
	}

	// Concurrently handle the HTTP requests
	ch = make(chan vclResponse, len(vclFiles))
	for _, vclPath := range vclFiles {
		wg.Add(1)
		go uploadVCL(vclPath, client)
	}
	wg.Wait()
	close(ch)

	// Process the aggregated results
	for vclFile := range ch {
		if vclFile.Error {
			fmt.Printf("\nWhoops, the file '%s' didn't upload because of the following error:\n\t%s\n", yellow(vclFile.Name), red(vclFile.Content))
		} else {
			fmt.Printf("\nYay, the file '%s' was uploaded successfully", green(vclFile.Name))
		}
	}
}

func getStatusVersion(statusVersion string, client *fastly.Client) (string, error) {
	versionStatus, err := client.GetVersion(&fastly.GetVersionInput{
		Service: fastlyServiceID,
		Version: statusVersion,
	})
	if err != nil {
		return "", err
	}

	status := green("not activated")
	if versionStatus.Active {
		status = red("already activated")
	}

	return status, nil
}

func cloneFromVersion(version string, client *fastly.Client) (*fastly.Version, error) {
	clonedVersion, err := client.CloneVersion(&fastly.CloneVersionInput{
		Service: fastlyServiceID,
		Version: version,
	})
	if err != nil {
		return nil, err
	}

	return clonedVersion, nil
}

func getLatestVCLVersion(client *fastly.Client) (string, error) {
	// We have to get all the versions and then sort them to find the actual latest
	listVersions, err := client.ListVersions(&fastly.ListVersionsInput{
		Service: fastlyServiceID,
	})
	if err != nil {
		return "", err
	}

	wv := wrappedVersions{}
	for _, v := range listVersions {
		i, err := strconv.Atoi(v.Number)
		if err != nil {
			return "", err
		}
		wv = append(wv, version{i, v})
	}
	sort.Sort(wv)

	return strconv.Itoa(wv[len(wv)-1].Number), nil
}

func aggregate(path string, f os.FileInfo, err error) error {
	if validPathDefaults(path) && validPathUserDefined(path) && !invalidPathUserDefined(path) {
		vclFiles = append(vclFiles, path)
	}

	return nil
}

func validPathDefaults(path string) bool {
	return !strings.Contains(path, ".git") && strings.Contains(path, ".vcl")
}

func validPathUserDefined(path string) bool {
	return dirMatchRegex.MatchString(path)
}

func invalidPathUserDefined(path string) bool {
	return dirSkipRegex.MatchString(path)
}

func uploadVCL(path string, client *fastly.Client) {
	defer wg.Done()

	name := extractName(path)
	content, err := getLocalVCL(path)

	if err != nil {
		ch <- vclResponse{
			Path:    path,
			Name:    name,
			Content: fmt.Sprintf("get local vcl error: %s", err),
			Error:   true,
		}
	} else {
		vclFile, err := client.CreateVCL(&fastly.CreateVCLInput{
			Service: fastlyServiceID,
			Version: selectedVersion,
			Name:    name,
			Content: content,
		})

		if err != nil {
			fmt.Printf("\nThere was an error creating the file '%s':\n%s\nWe'll now try updating this file instead of creating it\n", yellow(name), red(err))

			vclFileUpdate, updateErr := client.UpdateVCL(&fastly.UpdateVCLInput{
				Service: fastlyServiceID,
				Version: selectedVersion,
				Name:    name,
				Content: content,
			})
			if updateErr != nil {
				ch <- vclResponse{
					Path:    path,
					Name:    name,
					Content: fmt.Sprintf("error: %s", updateErr),
					Error:   true,
				}
			} else {
				ch <- vclResponse{
					Path:    path,
					Name:    name,
					Content: vclFileUpdate.Content,
					Error:   false,
				}
			}
		} else {
			ch <- vclResponse{
				Path:    path,
				Name:    name,
				Content: vclFile.Content,
				Error:   false,
			}
		}
	}
}

func extractName(path string) string {
	_, file := filepath.Split(path)
	return strings.Split(file, ".")[0]
}

func getLocalVCL(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func getLatestServiceVersion(client *fastly.Client) (string, string, error) {
	latestVersion, err := getLatestVCLVersion(client)
	if err != nil {
		return "", "", err
	}

	status, err := getStatusVersion(latestVersion, client)
	if err != nil {
		return "", "", err
	}

	return latestVersion, status, nil
}
