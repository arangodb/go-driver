//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
)

const (
	preReleasePrefix = "preview-"
)

var (
	versionFile string // Full path of VERSION file
	releaseType string // What type of release to create (major|minor|patch)

	ghRelease string // Full path of github-release tool
	ghUser    string // Github account name to create release in
	ghRepo    string // Github repository name to create release in

	preRelease bool // If set, mark release as preview
	dryRun     bool // If set, do not really push a release or any git changes
)

func init() {
	defaultDryRun := getBoolEnvVar("DRYRUN", false)
	flag.StringVar(&versionFile, "versionfile", "./VERSION", "Path of the VERSION file")
	flag.StringVar(&releaseType, "type", "patch", "Type of release to build (major|minor|patch)")
	flag.StringVar(&ghRelease, "github-release", "github-release", "Full path of github-release tool")
	flag.StringVar(&ghUser, "github-user", "arangodb", "Github account name to create release in")
	flag.StringVar(&ghRepo, "github-repo", "go-driver", "Github repository name to create release in")
	flag.BoolVar(&preRelease, "prerelease", false, "If set, mark release as preview")
	flag.BoolVar(&dryRun, "dryrun", defaultDryRun, "If set, do not really push a release or any git changes")
}

func main() {
	flag.Parse()
	ensureGithubToken()
	checkCleanRepo()

	version := bumpVersionInFile(releaseType)
	tagName := fmt.Sprintf("v%s", version)

	// create Tag
	gitTag(tagName)
	gitPush(tagName)

	// create release
	githubCreateDraftRelease(tagName)
	ghPublishDraftRelease(tagName)

	// bump version to devel
	bumpVersionInFile("devel")
	gitPush("")
}

// ensureGithubToken makes sure the GITHUB_TOKEN env var is set.
func ensureGithubToken() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		p := filepath.Join(os.Getenv("HOME"), ".arangodb/github-token")
		if raw, err := os.ReadFile(p); err != nil {
			log.Fatalf("Failed to release '%s': %v", p, err)
		} else {
			token = strings.TrimSpace(string(raw))
			os.Setenv("GITHUB_TOKEN", token)
		}
	}
}

func checkCleanRepo() {
	output, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		log.Fatalf("Failed to check git status: %v\n", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		log.Fatal("Repository has uncommitted changes\n")
	}
}

func runMake(target string) {
	if err := run("make", target); err != nil {
		log.Fatalf("Failed to make %s: %v\n", target, err)
	}
}

func bumpVersionInFile(action string) string {
	contents, err := os.ReadFile(versionFile)
	if err != nil {
		log.Fatalf("Cannot read '%s': %v\n", versionFile, err)
	}

	version := semver.New(strings.TrimSpace(string(contents)))
	bumpVersion(version, action, preRelease)

	contents = []byte(version.String())
	if err := os.WriteFile(versionFile, contents, 0755); err != nil {
		log.Fatalf("Cannot write '%s': %v\n", versionFile, err)
	}

	gitCommitAll(fmt.Sprintf("Updated to %s", version))
	log.Printf("Updated '%s' to '%s'\n", versionFile, string(contents))

	return version.String()
}

func bumpVersion(version *semver.Version, action string, isPreRelease bool) {
	if action == "devel" {
		version.Metadata = "git"
		return
	}
	version.Metadata = ""

	currVersionIsPreRelease := strings.HasPrefix(string(version.PreRelease), preReleasePrefix)
	if isPreRelease {
		firstPreRelease := semver.PreRelease(fmt.Sprintf("%s1", preReleasePrefix))
		switch action {
		case "patch":
			if currVersionIsPreRelease {
				version.PreRelease = bumpPreReleaseIndex(version.PreRelease)
			} else {
				version.BumpPatch()
				version.PreRelease = firstPreRelease
			}
		case "minor":
			if currVersionIsPreRelease && version.Patch == 0 {
				version.PreRelease = bumpPreReleaseIndex(version.PreRelease)
			} else {
				version.BumpMinor()
				version.PreRelease = firstPreRelease
			}
		case "major":
			if currVersionIsPreRelease && version.Minor == 0 && version.Patch == 0 {
				version.PreRelease = bumpPreReleaseIndex(version.PreRelease)
			} else {
				version.BumpMajor()
				version.PreRelease = firstPreRelease
			}
		}
	} else {
		version.PreRelease = ""
		if !currVersionIsPreRelease {
			switch action {
			case "patch":
				version.BumpPatch()
			case "minor":
				version.BumpMinor()
			case "major":
				version.BumpMajor()
			}
		}
	}
}

func bumpPreReleaseIndex(preReleaseStr semver.PreRelease) semver.PreRelease {
	indexStr := strings.TrimPrefix(string(preReleaseStr), preReleasePrefix)
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		log.Fatalf("Could not parse prerelease: %s", preReleaseStr)
	}
	index++
	return semver.PreRelease(fmt.Sprintf("%s%d", preReleasePrefix, index))
}

func gitCommitAll(message string) {
	if dryRun {
		log.Printf("Skipping git commit with message '%s'", message)
	} else {
		args := []string{
			"commit",
			"--all",
			"-m", message,
		}
		if err := run("git", args...); err != nil {
			log.Fatalf("Failed to commit: %v\n", err)
		}
	}
}

func gitPush(tag string) {
	if dryRun {
		log.Printf("Skipping git push")
	} else {
		if err := run("git", "push", "-u", "origin", "HEAD"); err != nil {
			log.Fatalf("Failed to push: %v\n", err)
		}
		if tag != "" {
			if err := run("git", "push", "origin", tag); err != nil {
				log.Fatalf("Failed to push tag: %v\n", err)
			}
		}
	}
}

func gitTag(tagName string) {
	if dryRun {
		log.Printf("Skipping git tag with name '%s'", tagName)
	} else {
		if err := run("git", "tag", tagName); err != nil {
			log.Fatalf("Failed to tag: %v\n", err)
		}
	}
}

func githubCreateDraftRelease(tagName string) {
	if dryRun {
		log.Printf("Skipping github release with tag name '%s'", tagName)
	} else {
		// Create draft release
		args := []string{
			"release",
			"--user", ghUser,
			"--repo", ghRepo,
			"--tag", tagName,
			"--draft",
		}
		if preRelease {
			args = append(args, "--pre-release")
		}
		if err := run(ghRelease, args...); err != nil {
			log.Fatalf("Failed to create github release: %v\n", err)
		}
		// Ensure release created (sometimes there is a delay between creation request and it's availability for assets upload)
		ensureReleaseCreated(tagName)
	}
}

func ghPublishDraftRelease(tagName string) {
	if dryRun {
		log.Printf("Skipping github release finalize with tag name '%s'", tagName)
		return
	}
	args := []string{
		"edit",
		"--user", ghUser,
		"--repo", ghRepo,
		"--tag", tagName,
	}
	if preRelease {
		args = append(args, "--pre-release")
	}
	if err := run(ghRelease, args...); err != nil {
		log.Fatalf("Failed to finalize github release: %v\n", err)
	}
	log.Printf("Release published to GitHub: https://github.com/%s/%s/releases/tag/%s", ghUser, ghRepo, tagName)

	log.Printf("Add changelog to release manually and publish it from GitHub")
}

func ensureReleaseCreated(tagName string) {
	const attemptsCount = 5
	var interval = time.Second
	var err error

	for i := 1; i <= attemptsCount; i++ {
		time.Sleep(interval)
		interval *= 2

		args := []string{
			"info",
			"--user", ghUser,
			"--repo", ghRepo,
			"--tag", tagName,
		}
		err = run(ghRelease, args...)
		if err == nil {
			return
		}
		log.Printf("attempt #%d to get release info for tag %s failed. Retry in %s...", i, tagName, interval.String())
	}

	log.Fatalf("failed to get release info for tag %s", tagName)
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// getEnvVar returns the value of the environment variable with given key of the given default
// value of no such variable exist or is empty.
func getEnvVar(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnvVar returns the bool value of the environment variable with given key of the given default
// value of no such variable exist or is empty.
func getBoolEnvVar(key string, defaultValue bool) bool {
	value := getEnvVar(key, strconv.FormatBool(defaultValue))
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	return defaultValue
}
