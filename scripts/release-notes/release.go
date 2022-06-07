package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	r := parseArgs()
	releaseNotes := generateReleaseNotes(r.gitHubToken, r.version)

	f, err := os.Create("out.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(releaseNotes)
}

func parseArgs() *release {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")

	version := flag.String("version", "", "Release version tag. Must be a valid semver. The branch is automatically detected from the major and minor release.")
	githubToken := flag.String("github-token", "", "Github Token.")
	flag.Parse()

	if *githubToken == "" {
		log.Fatal("-github-token is a required argument")
	} else if *version == "" {
		log.Fatal("-version is a required argument")
	}

	return &release{
		version:     *version,
		gitHubToken: *githubToken,
	}
}
