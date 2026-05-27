package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Device struct {
	serial string
	status string
}

type FlashablePartition struct {
	partition string
	imagePath string
}

func checkFastboot() bool {
	_, err := exec.LookPath("fastboot")
	return err == nil
}

func getFastbootDevices() ([]Device, error) {
	cmd := exec.Command("fastboot", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var devices []Device
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "fastboot" {
			devices = append(devices, Device{serial: fields[0], status: fields[1]})
		}
	}
	return devices, nil
}

func getFlashablePartitions(dirName string) ([]FlashablePartition, error) {
	images, err := os.ReadDir(dirName)
	if err != nil {
		return nil, err
	}
	var flashablePartitions []FlashablePartition
	for _, img := range images {
		if strings.HasSuffix(img.Name(), ".img") {
			flashablePartitions = append(
				flashablePartitions,
				FlashablePartition{
					partition: strings.TrimSuffix(img.Name(), ".img"),
					imagePath: filepath.Join(dirName, img.Name()),
				},
			)
		}
	}
	return flashablePartitions, nil
}

func flashHandler(dirName string) {
	fmt.Printf("Flashing extracted images to the device on current active slot\n")
	devices, err := getFastbootDevices()
	if err != nil {
		log.Fatalf("Failed to get connected devices in fastboot mode: %s\n", err)
	}
	if len(devices) == 0 {
		fmt.Printf("No devices found in fastboot mode\n")
		return
	}
	if len(devices) > 1 {
		//TODO : Handle multiple devices connected in fastboot mode with serial
		fmt.Printf("Multiple devices detected. Please connect only one device in fastboot mode\n")
		return
	}
	flashablePartitions, err := getFlashablePartitions(dirName)
	if err != nil {
		log.Fatalf("Failed to get flashable partitions: %s\n", err)
	}
	for _, partition := range flashablePartitions {
		fmt.Printf("Flashing %s to %s partition\n", partition.imagePath, partition.partition)
		err := flashImage(partition.imagePath, partition.partition)
		if err != nil {
			fmt.Printf("Failed to flash %s to %s partition: %s\n", partition.imagePath, partition.partition, err)
		}
	}
	fmt.Printf("Flashing completed successfully\n")
}

func flashImage(imagePath string, partition string) error {
	cmd := exec.Command("fastboot", "flash", partition, imagePath)
	return cmd.Run()
}
