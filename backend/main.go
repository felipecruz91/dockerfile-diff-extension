package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/labstack/echo/v4/middleware"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type DiffResponse struct {
	Image1 Image `json:"image1"`
	Image2 Image `json:"image2"`
}

type Image struct {
	Name       string `json:"name"`
	Dockerfile string `json:"dockerfile"`
}

var logger = logrus.New()

func main() {
	logger.SetOutput(os.Stdout)

	var socketPath string
	flag.StringVar(&socketPath, "socket", "/run/guest-services/backend.sock", "Unix domain socket to listen on")
	flag.Parse()

	_ = os.RemoveAll(socketPath)

	logger.SetOutput(os.Stdout)

	logMiddleware := middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: middleware.DefaultSkipper,
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
			`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
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

type result struct {
	image      string
	dockerfile string
}

func doDiff(ctx echo.Context) error {
	image1 := ctx.QueryParam("image1")
	image2 := ctx.QueryParam("image2")

	var wg sync.WaitGroup
	c := make(chan result, 2)

	for _, image := range []string{image1, image2} {
		image := image
		wg.Add(1)
		go func() {
			defer wg.Done()

			dockerfile := getDockerfile(ctx.Request().Context(), image)

			logrus.Infof("Sending msg from image %s to channel", image)
			c <- result{image, dockerfile}
		}()
	}

	logrus.Info("Waiting for goroutines to complete...")
	wg.Wait()

	close(c)

	dr := &DiffResponse{}

	logrus.Info("Ready to receive msgs from channel...")
	for x := range c {
		logrus.Infof("Received msg from channel: %s - %s...", x.image, x.dockerfile[:5])
		if x.image == image1 {
			dr.Image1 = Image{
				Name:       x.image,
				Dockerfile: x.dockerfile,
			}
		} else {
			dr.Image2 = Image{
				Name:       x.image,
				Dockerfile: x.dockerfile,
			}
		}
	}

	return ctx.JSON(http.StatusOK, dr)
}
