package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type SlimReport struct {
	ImageStack []struct {
		IsTopImage   bool      `json:"is_top_image"`
		ID           string    `json:"id"`
		FullName     string    `json:"full_name"`
		RepoName     string    `json:"repo_name"`
		VersionTag   string    `json:"version_tag"`
		RawTags      []string  `json:"raw_tags"`
		CreateTime   time.Time `json:"create_time"`
		NewSize      int       `json:"new_size"`
		NewSizeHuman string    `json:"new_size_human"`
		Instructions []struct {
			Type              string    `json:"type"`
			Time              time.Time `json:"time"`
			IsNop             bool      `json:"is_nop"`
			LocalImageExists  bool      `json:"local_image_exists"`
			LayerIndex        int       `json:"layer_index"`
			LayerID           string    `json:"layer_id"`
			LayerFsdiffID     string    `json:"layer_fsdiff_id"`
			Size              int       `json:"size"`
			SizeHuman         string    `json:"size_human,omitempty"`
			Params            string    `json:"params,omitempty"`
			CommandSnippet    string    `json:"command_snippet"`
			CommandAll        string    `json:"command_all"`
			Target            string    `json:"target,omitempty"`
			SourceType        string    `json:"source_type,omitempty"`
			IsExecForm        bool      `json:"is_exec_form,omitempty"`
			EmptyLayer        bool      `json:"empty_layer,omitempty"`
			SystemCommands    []string  `json:"system_commands,omitempty"`
			IsLastInstruction bool      `json:"is_last_instruction,omitempty"`
			RawTags           []string  `json:"raw_tags,omitempty"`
		} `json:"instructions"`
	} `json:"image_stack"`
}

// getDockerfile will output the reverse-engineered Dockerfile from an image.
func getDockerfile(ctx context.Context, image string) string {
	replacer := strings.NewReplacer("/", "_", ":", "_")
	reportFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-slim.report.json", replacer.Replace(image)))
	logger.Infof("reportFile: %s", reportFile)
	defer cleanup(reportFile)

	//TODO: Pull private images from a registry using the `--pull` flag using the docker credentials.
	cmd := exec.CommandContext(ctx, "slim", "--report", reportFile, "xray", "--target", image, "--changes", "all", "--changes-output", "report")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}

	sr := &SlimReport{}
	reportOutput, err := os.ReadFile(reportFile)
	if err = json.Unmarshal(reportOutput, sr); err != nil {
		logger.Fatal(err)
	}

	var sb strings.Builder

	for _, is := range sr.ImageStack {
		for _, i := range is.Instructions {
			fmt.Fprintf(&sb, "%s\n", i.CommandAll)
		}
	}

	return sb.String()
}

func cleanup(reportFile string) {
	if _, err := os.Stat(reportFile); err == nil {
		logger.Infof("Removing file %s", reportFile)
		cmd := exec.Command("rm", "-rf", reportFile)
		if err := cmd.Run(); err != nil {
			logger.Fatal(err)
		}
	}
}
