package main

import (
	"bytes"
	"flag"
	"github.com/fsouza/go-dockerclient"
	"log"
	"net/http"
	"os"
	"strings"
)

func RemoveContainerAndImage(client *docker.Client, imageTag string) {
	log.Printf("Removing previous containers and images for %s\n", imageTag)
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})
	for _, cnt := range containers {
		if cnt.Image == imageTag {
			log.Printf("  > Found container ID %s\n", cnt.ID)
			client.RemoveContainer(docker.RemoveContainerOptions{ID: cnt.ID, Force: true})
		}
	}

	imgs, _ := client.ListImages(docker.ListImagesOptions{All: false})
	for _, img := range imgs {
		if img.RepoTags[0] == imageTag {
			log.Printf("  > Found image ID %s\n", img.ID)
			client.RemoveImage(img.ID)
		}
	}

	log.Println("Done.")
}

func AnnounceContainer(appName string, branchName string, ip string, etcdBaseUrl string) error {
	content := "value=" + ip
	client := &http.Client{}
	url := etcdBaseUrl + appName + "/" + branchName
	request, err := http.NewRequest("PUT", url, strings.NewReader(content))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.ContentLength = int64(len(content))
	_, err = client.Do(request)

	return err
}

func BuildImage(client *docker.Client, imgTag string, path string) {
	log.Println("Building the new image...")
	var buf bytes.Buffer
	opts := docker.BuildImageOptions{
		Name:                imgTag,
		NoCache:             true,
		SuppressOutput:      false,
		RmTmpContainer:      true,
		ForceRmTmpContainer: true,
		OutputStream:        &buf,
		ContextDir:          path,
	}
	err := client.BuildImage(opts)
	if err != nil {
		log.Fatalf("Failed building image for %s\n", imgTag)
	}
	log.Println("Done.")
}

func CreateAndStartContainer(client *docker.Client, imgTag string) string {
	log.Println("Creating container...")
	config := docker.Config{Image: imgTag}
	container, err := client.CreateContainer(docker.CreateContainerOptions{Config: &config})

	if err != nil {
		log.Fatalf("Failed for image %s: %v\n", imgTag, err)
	}
	log.Printf("  > created container %s\n", container.ID)
	log.Println("Done")

	client.StartContainer(container.ID, nil)
	containerInfo, err := client.InspectContainer(container.ID)
	if err != nil {
		log.Fatalf("Failed getting container info for running container %s: %v\n", container.ID, err)
	}
	log.Printf("Container running at %s\n", containerInfo.NetworkSettings.IPAddress)
	return containerInfo.NetworkSettings.IPAddress
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 3 {
		log.Println("Usage: godeploy <path> <app name> <branch name>")
		os.Exit(0)
	}

	imgTag := args[1] + ":" + args[2]

	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)

	RemoveContainerAndImage(client, imgTag)
	BuildImage(client, imgTag, args[0])

	ip := CreateAndStartContainer(client, imgTag)

	log.Println("Announcing container to etcd...")
	if err := AnnounceContainer(args[1], args[2], ip, "http://localhost:2379/v2/keys/deployments/"); err != nil {
		log.Fatalf("Failed announcing container for %s (branch %s) to etcd at %s: %v\n", args[1], args[2], "http://localhost:2379/v2/keys/deployments/", err)
	}

	log.Println("Announced container.")
}
