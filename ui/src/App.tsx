import React, { useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { TextField } from "@mui/material";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import { blue, green } from "@mui/material/colors";
import Fab from "@mui/material/Fab";
import CheckIcon from "@mui/icons-material/Check";
import DifferenceIcon from "@mui/icons-material/Difference";
import Grid from "@mui/material/Grid";

// Note: This line relies on Docker Desktop's presence as a host application.
// If you're running this React app in a browser, it won't work properly.
const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export function App() {
  const [response, setResponse] = useState("");
  const types = ["Split", "Unified"];
  const [active, setActive] = useState(types[0]);
  const [image1, setImage1] = useState("citizenstig/httpbin");
  const [image2, setImage2] = useState("kennethreitz/httpbin");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);

  const buttonSx = {
    ...(success && {
      bgcolor: green[500],
      "&:hover": {
        bgcolor: green[700],
      },
    }),
  };

  const ddClient = useDockerDesktopClient();

  async function pullImageIfNotPresent(image: string) {
    console.log(`Checking if image ${image} is present...`);

    let success = false;
    await ddClient.docker.cli
      .exec("inspect", [image])
      .then(() => (success = true))
      .catch(async () => {
        console.log(`Image ${image} not present`);
        success = await pullImage(image);
      });

    return success;
  }

  async function pullImage(image: any) {
    console.log(`Pulling image ${image}...`);

    let success = false;

    await ddClient.docker.cli
      .exec("pull", [image])
      .then(() => {
        console.log(`Image ${image} pulled successfully`);
        success = true;
      })
      .catch((error: any) => {
        ddClient.desktopUI.toast.error(
          `Failed to pull image ${image1}: ${error}`
        );
      });

    console.log(`success: ${success}`);

    return success;
  }

  const get = async () => {
    if (!loading) {
      setSuccess(false);
      setLoading(true);
    }

    ddClient.desktopUI.toast.warning(
      "Analyzing images, this may take a few seconds..."
    );

    const i1r = await pullImageIfNotPresent(image1);
    const i2r = await pullImageIfNotPresent(image2);

    console.log(`image 1 ready? ${i1r}`);
    console.log(`image 2 ready? ${i2r}`);

    if (!i1r || !i2r) {
      setSuccess(false);
      setLoading(false);
      return;
    }

    console.log("Analyzing images...");
    const result = await ddClient.extension.vm?.service
      ?.get(`/diff?image1=${image1}&image2=${image2}`)
      .catch((error) => {
        ddClient.desktopUI.toast.error(`Something went wrong: ${error}`);
      });

    //@ts-ignore
    console.log(result.image1.dockerfile);
    //@ts-ignore
    console.log(result.image2.dockerfile);
    //@ts-ignore
    setResponse(result);

    setSuccess(true);
    setLoading(false);

    setTimeout(() => {
      setSuccess(false); // to show the compare button again after the success
    }, 3000);
  };

  const handleChange = (_: any, newActive: any) => {
    setActive(newActive);
  };

  return (
    <div className="App">
      <h1>Dockerfile Diff</h1>
      <h2>Compare the Dockerfile of 2 images and find their differences.</h2>

      <Box sx={{ flexGrow: 1 }}>
        <Grid
          container
          direction="row"
          justifyContent="center"
          alignItems="center"
        >
          <Grid item xs={4}>
            <TextField
              required
              id="image-1"
              label="Image"
              placeholder="docker.io/felipe/image:latest"
              onChange={(event) => {
                const { value } = event.target;
                setImage1(value);
              }}
              margin="normal"
              value={image1}
            />
          </Grid>
          <Grid item xs={0}>
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Box sx={{ m: 1, position: "relative" }}>
                <Fab
                  aria-label="compare"
                  color="primary"
                  sx={buttonSx}
                  onClick={get}
                >
                  {success ? <CheckIcon /> : <DifferenceIcon />}
                </Fab>
                {loading && (
                  <CircularProgress
                    size={68}
                    sx={{
                      color: blue[500],
                      position: "absolute",
                      top: -6,
                      left: -6,
                      zIndex: 1,
                    }}
                  />
                )}
              </Box>
            </Box>
          </Grid>
          <Grid item xs={4}>
            <TextField
              required
              id="image-2"
              label="Image"
              placeholder="ghcr.io/felipe/image:latest"
              onChange={(event) => {
                const { value } = event.target;
                setImage2(value);
              }}
              margin="normal"
              value={image2}
            />
          </Grid>
        </Grid>
      </Box>

      {response && (
        <Box sx={{ width: "auto" }}>
          <ToggleButtonGroup
            color="primary"
            value={active}
            exclusive
            onChange={handleChange}
          >
            {types.map((type) => (
              <ToggleButton
                key={type}
                value={type}
                onClick={() => setActive(type)}
              >
                {type}
              </ToggleButton>
            ))}
          </ToggleButtonGroup>
          <ReactDiffViewer
            compareMethod={DiffMethod.WORDS}
            // @ts-ignore
            oldValue={response.image1.dockerfile}
            // @ts-ignore
            newValue={response.image2.dockerfile}
            splitView={active === "Split"}
            // @ts-ignore
            leftTitle={`Dockerfile (${response.image1.name})`}
            // @ts-ignore
            rightTitle={`Dockerfile (${response.image2.name})`}
          />
        </Box>
      )}
    </div>
  );
}
