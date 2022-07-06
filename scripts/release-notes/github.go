package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/golang/glog"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/release-notes/git"
)

const additionalResources = `Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- [License][license]

License: https://github.com/kubevirt/must-gather/blob/main/LICENSE

---
`

func generateReleaseNotes(gitHubToken string, version string) string {
	var releaseNotes strings.Builder
	project := initProject(gitHubToken, version)

	releaseNotes.WriteString(header(project))
	releaseNotes.WriteString(notableChanges(project))
	releaseNotes.WriteString(contributors(project))

	releaseNotes.WriteString(additionalResources)

	return releaseNotes.String()
}

func initProject(gitHubToken string, version string) *git.Project {
	project := git.InitProject(gitOwner, name, name, gitTargetDir, version, gitHubToken)

	err := project.CheckoutUpstream()
	if err != nil {
		log.Fatalf("ERROR checking out upstream: %s\n", err)
	}

	err = project.CheckCurrentTagExists()
	if err != nil {
		log.Fatalf("ERROR verifying tag exists: %s\n", err)
	}

	err = project.VerifySemverTag()
	if err != nil {
		log.Fatalf("ERROR requested tag invalid: %s\n", err)
	}

	return project
}

func header(project *git.Project) string {
	var builder strings.Builder

	tagUrl := fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", gitOwner, name, project.CurrentTag)

	numChanges, err := project.GetNumChanges()
	if err != nil {
		glog.Fatalf("ERROR failed to get num changes: %s\n", err)
	}

	typeOfChanges, err := project.GetTypeOfChanges()
	if err != nil {
		glog.Fatalf("ERROR failed to get type of changes: %s\n", err)
	}

	builder.WriteString(fmt.Sprintf("This release follows %s and consists of %d changes, leading to %s.\n\n", project.PreviousTag, numChanges, typeOfChanges))
	builder.WriteString(fmt.Sprintf("The source code and selected binaries are available for download at: %s.\n\n", tagUrl))
	builder.WriteString(fmt.Sprintf("The primary release artifact of %s is the git tree. The release tag is\n", name))
	builder.WriteString(fmt.Sprintf("signed and can be verified using `git tag -v %s`.\n\n", project.CurrentTag))
	builder.WriteString(fmt.Sprintf("Pre-built containers are published on Quay and can be viewed at: <https://quay.io/%s/>.\n\n", quayOwner))

	return builder.String()
}

func notableChanges(project *git.Project) string {
	var builder strings.Builder

	builder.WriteString("Notable changes\n---------------\n\n")

	releaseNotes, err := project.GetReleaseNotes()
	if err != nil {
		glog.Fatalf("ERROR failed to get release notes of %s: %s\n", project.Name, err)
	}

	builder.WriteString(fmt.Sprintf("### %s - %s\n", project.Name, project.CurrentTag))
	if len(releaseNotes) > 0 {
		for _, note := range releaseNotes {
			builder.WriteString(fmt.Sprintf("- %s\n", note))
		}
	} else {
		builder.WriteString("No notable changes\n")
	}

	builder.WriteString("\n")

	return builder.String()
}

func contributors(project *git.Project) string {
	var builder strings.Builder

	contributorList, err := project.GetContributors()
	if err != nil {
		glog.Fatalf("ERROR failed to get contributor list: %s\n", err)
	}

	var sb strings.Builder
	numContributors := 0
	for _, contributor := range contributorList {
		if len(contributor) != 0 {
			numContributors++
			sb.WriteString(fmt.Sprintf(" - %s\n", strings.TrimSpace(contributor)))
		}
	}

	builder.WriteString("\nContributors\n------------\n")
	builder.WriteString(fmt.Sprintf("%d people contributed to this release:\n\n", numContributors))
	builder.WriteString(sb.String())
	builder.WriteString("\n")

	return builder.String()
}
