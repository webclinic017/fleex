package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sw33tLie/fleex/pkg/box"
	"github.com/sw33tLie/fleex/pkg/sshutils"
	"github.com/sw33tLie/fleex/pkg/utils"
)

// SpawnFleet spawns a DigitalOcean fleet
func SpawnFleet(fleetName string, fleetCount int, image string, region string, size string, sshFingerprint string, tags []string, token string, wait bool) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	droplets := []string{}

	for i := 0; i < fleetCount; i++ {
		droplets = append(droplets, fleetName+"-"+strconv.Itoa(i+1))
	}

	// my image: 86085763
	var createRequest *godo.DropletMultiCreateRequest
	imageIntID, err := strconv.Atoi(image)
	if err != nil {
		fmt.Println("err:", image)
		createRequest = &godo.DropletMultiCreateRequest{
			Names:    droplets,
			Region:   region,
			Size:     size,
			UserData: "echo 'root:1337superPass' | chpasswd",
			Image: godo.DropletCreateImage{
				Slug: image,
			},
			SSHKeys: []godo.DropletCreateSSHKey{
				{Fingerprint: sshFingerprint},
			},
			Tags: tags,
		}
	} else {
		fmt.Println("err:", image)
		createRequest = &godo.DropletMultiCreateRequest{
			Names:    droplets,
			Region:   region,
			Size:     size,
			UserData: "echo 'root:1337superPass' | chpasswd",
			Image: godo.DropletCreateImage{
				ID: imageIntID,
			},
			SSHKeys: []godo.DropletCreateSSHKey{
				{Fingerprint: sshFingerprint},
			},
			Tags: tags,
		}
	}

	_, _, err = client.Droplets.CreateMultiple(ctx, createRequest)

	if err != nil {
		utils.Log.Fatal(err)
	}

	if wait {
		for {
			stillNotReady := false
			fleet := GetFleet(fleetName, token)
			if len(fleet) == fleetCount {
				for i := range fleet {
					if fleet[i].Status != "active" {
						stillNotReady = true
					}
				}
			}

			if stillNotReady {
				time.Sleep(8 * time.Second)
			} else {
				break
			}
		}
	}

}

func GetFleet(fleetName, token string) (fleet []box.Box) {
	boxes := GetBoxes(token)

	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			fleet = append(fleet, box)
		}
	}
	return fleet
}

func GetBoxes(token string) (boxes []box.Box) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		utils.Log.Fatal(err)
	}

	for _, d := range droplets {
		ip, _ := d.PublicIPv4()
		boxes = append(boxes, box.Box{ID: d.ID, Label: d.Name, Group: "", Status: d.Status, IP: ip})
	}
	return boxes
}

func ListBoxes(token string) {
	boxes := GetBoxes(token)
	for _, box := range boxes {
		fmt.Println(box.ID, box.Label, box.Group, box.Status, box.IP)
	}
}

func DeleteFleet(name string, token string) {
	droplets := GetBoxes(token)
	for _, droplet := range droplets {
		if droplet.Label == name {
			// It's a single box
			deleteBoxByID(droplet.ID, token)
			return
		}
	}

	// Otherwise, we got a fleet to delete
	for _, droplet := range droplets {
		if strings.HasPrefix(droplet.Label, name) {
			deleteBoxByID(droplet.ID, token)
		}
	}
}

func ListImages(token string) {
	// TODO
	client := godo.NewFromToken(token)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 9999,
	}

	images, _, err := client.Images.ListUser(ctx, opt)
	if err != nil {
		utils.Log.Fatal(err)
	}
	for _, image := range images {
		fmt.Println(image.ID, image.Name, image.Status, image.SizeGigaBytes)
	}
}

func deleteBoxByID(ID int, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, err := client.Droplets.Delete(ctx, ID)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func deleteBoxByTag(tag string, token string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	_, err := client.Droplets.DeleteByTag(ctx, tag)
	if err != nil {
		utils.Log.Fatal(err)
	}
}

func CountFleet(fleetName string, boxes []box.Box) (count int) {
	for _, box := range boxes {
		if strings.HasPrefix(box.Label, fleetName) {
			count++
		}
	}
	return count
}

func RunCommand(name, command string, port int, username, password, token string) {
	//doSshUser := viper.GetString("digitalocean.username")
	//doSshPort := viper.GetInt("digitalocean.port")
	// doSshPassword := viper.GetString("digitalocean.password")
	boxes := GetBoxes(token)

	// fmt.Println(port, username, password)

	for _, box := range boxes {
		if box.Label == name {
			// It's a single box
			boxIP := box.IP
			sshutils.RunCommand(command, boxIP, port, username, password)
			return
		}
	}

	// Otherwise, send command to a fleet
	fleetSize := CountFleet(name, boxes)

	fleet := make(chan *box.Box, fleetSize)
	processGroup := new(sync.WaitGroup)
	processGroup.Add(fleetSize)

	for i := 0; i < fleetSize; i++ {
		go func() {
			for {
				box := <-fleet

				if box == nil {
					break
				}
				boxIP := box.IP
				sshutils.RunCommand(command, boxIP, port, username, password)
			}
			processGroup.Done()
		}()
	}

	for i := range boxes {
		if strings.HasPrefix(boxes[i].Label, name) {
			fleet <- &boxes[i]
		}
	}

	close(fleet)
	processGroup.Wait()
}

func RunCommandByIP(ip, command string, port int, username, password, token string) {
	sshutils.RunCommand(command, ip, port, username, password)
}

func CreateImage(token string, diskID int, label string) {
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	action, _, err := client.DropletActions.Snapshot(ctx, diskID, label)
	if err != nil {
		utils.Log.Fatal(err)
	}
	fmt.Println(action)
}
