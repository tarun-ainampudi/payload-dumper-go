package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type device struct {
	serial string
	status string
}

type flashable_partition struct {
	partition  string
	image_path string
}

func check_fastboot() bool {
	_, err := exec.LookPath("fastboot")
	return err == nil
}

func get_devices() ([]device, error) {
	cmd := exec.Command("fastboot", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var devices []device
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "fastboot" {
			devices = append(devices, device{serial: fields[0], status: fields[1]})
		}
	}
	return devices, nil
}

func get_flashable_partitions(extracted_path string) ([]flashable_partition, error) {
	images, err := os.ReadDir(extracted_path)
	if err != nil {
		return nil, err
	}
	var flashable_partitions []flashable_partition
	for _, img := range images {
		if strings.HasSuffix(img.Name(), ".img") {
			flashable_partitions = append(
				flashable_partitions,
				flashable_partition{
					partition:  strings.TrimSuffix(img.Name(), ".img"),
					image_path: filepath.Join(extracted_path, img.Name()),
				},
			)
		}
	}
	return flashable_partitions, nil
}

func flash_handler(extracted_path string) {
	fmt.Printf("Flashing extracted images to the device on current active slot\n")
	devices, err := get_devices()
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
	flashable_partitions, err := get_flashable_partitions(extracted_path)
	if err != nil {
		log.Fatalf("Failed to get flashable partitions: %s\n", err)
	}
	for _, partition := range flashable_partitions {
		fmt.Printf("Flashing %s to %s partition\n", partition.image_path, partition.partition)
		err := flash_image(partition.image_path, partition.partition)
		if err != nil {
			fmt.Printf("Failed to flash %s to %s partition: %s\n", partition.image_path, partition.partition, err)
		}
	}
	fmt.Printf("Flashing completed successfully\n")
}

func flash_image(image_path string, partition string) error {
	cmd := exec.Command("fastboot", "flash", partition, image_path)
	return cmd.Run()
}
