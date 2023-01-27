package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/middleware"

	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func main() {
	logrus.SetOutput(os.Stdout)

	var socketPath string
	flag.StringVar(&socketPath, "socket", "/run/guest-services/backend.sock", "Unix domain socket to listen on")
	flag.Parse()

	_ = os.RemoveAll(socketPath)

	logger.SetOutput(os.Stdout)

	logMiddleware := middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: middleware.DefaultSkipper,
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"` +
			`}` + "\n",
		CustomTimeFormat: "2006-01-02 15:04:05.00000",
		Output:           logger.Writer(),
	})

	logger.Infof("Starting listening on %s\n", socketPath)
	router := echo.New()
	router.HideBanner = true
	router.Use(logMiddleware)
	startURL := ""

	ln, err := listen(socketPath)
	if err != nil {
		logger.Fatal(err)
	}
	router.Listener = ln

	router.GET("/diff", doDiff)

	logger.Fatal(router.Start(startURL))
}

func listen(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

func doDiff(ctx echo.Context) error {
	image1 := ctx.QueryParam("image1")
	image2 := ctx.QueryParam("image2")

	logrus.Infof("image1: %s\n", image1)
	logrus.Infof("image2: %s\n", image2)

	dockerfile1 := GetDockerfile(image1)
	logrus.Infof("********** Dockerfile (%s) *************\n", image1)
	logrus.Info(dockerfile1)
	logrus.Info("****************************************")

	logrus.Infof("********** Dockerfile (%s) *************\n", image2)
	dockerfile2 := GetDockerfile(image2)
	logrus.Info(dockerfile2)
	logrus.Info("****************************************")

	dr := &DiffResponse{
		Image1: Image{
			Name:       image1,
			Dockerfile: dockerfile1,
		},
		Image2: Image{
			Name:       image2,
			Dockerfile: dockerfile2,
		},
	}

	return ctx.JSON(http.StatusOK, dr)
}

type DiffResponse struct {
	Image1 Image `json:"image1"`
	Image2 Image `json:"image2"`
}

type Image struct {
	Name       string `json:"name"`
	Dockerfile string `json:"dockerfile"`
}

type HTTPMessageBody struct {
	Message string
}

const reportFile = "slim.report.json"

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

// GetDockerfile will output the reverse-engineered Dockerfile from a **local** given image.
func GetDockerfile(image string) string {
	defer cleanup()

	ctx := context.TODO()
	//TODO: Be able to pull images from a registry using the `--pull` flag and sharing the docker credentials.
	cmd := exec.CommandContext(ctx, "docker-slim", "xray", "--target", image, "--changes", "all", "--changes-output", "report")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	sr := &SlimReport{}
	reportOutput, err := os.ReadFile(reportFile)
	if err = json.Unmarshal(reportOutput, sr); err != nil {
		log.Fatal(err)
	}

	var sb strings.Builder

	for _, is := range sr.ImageStack {
		for _, i := range is.Instructions {
			fmt.Fprintf(&sb, "%s\n", i.CommandAll)
		}
	}

	return sb.String()
}

func cleanup() {
	if _, err := os.Stat(reportFile); err == nil {
		logrus.Infof("Removing file %q\n", reportFile)
		cmd := exec.Command("rm", "-rf", reportFile)
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
