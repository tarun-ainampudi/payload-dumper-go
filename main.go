package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

func extractPayloadBin(filename string) string {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		log.Fatalf("Not a valid zip archive: %s\n", filename)
	}
	defer zipReader.Close()

	for _, file := range zipReader.Reader.File {
		if file.Name == "payload.bin" && file.UncompressedSize64 > 0 {
			zippedFile, err := file.Open()
			if err != nil {
				log.Fatalf("Failed to read zipped file: %s\n", file.Name)
			}

			tempfile, err := os.CreateTemp(os.TempDir(), "payload_*.bin")
			if err != nil {
				log.Fatalf("Failed to create a temp file located at %s\n", tempfile.Name())
			}
			defer tempfile.Close()

			_, err = io.Copy(tempfile, zippedFile)
			if err != nil {
				log.Fatal(err)
			}

			return tempfile.Name()
		}
	}

	return ""
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var (
		list            bool
		partitions      string
		extractRecovery bool
		outputDirectory string
		concurrency     int
		flash           bool
	)

	flag.IntVar(&concurrency, "c", runtime.NumCPU()/2, "Number of multiple workers to extract (shorthand)")
	flag.IntVar(&concurrency, "concurrency", runtime.NumCPU()/2, "Number of multiple workers to extract")
	flag.BoolVar(&list, "l", false, "Show list of partitions in payload.bin (shorthand)")
	flag.BoolVar(&list, "list", false, "Show list of partitions in payload.bin")
	flag.StringVar(&outputDirectory, "o", "", "Set output directory (shorthand)")
	flag.StringVar(&outputDirectory, "output", "", "Set output directory")
	flag.StringVar(&partitions, "p", "", "Dump only selected partitions (comma-separated) (shorthand)")
	flag.StringVar(&partitions, "partitions", "", "Dump only selected partitions (comma-separated)")
	flag.BoolVar(&extractRecovery, "recovery", false, "Extract boot,dtbo and vendor_boot images")
	flag.BoolVar(&extractRecovery, "r", false, "Extract boot,dtbo and vendor_boot images (shorthand)")
	flag.BoolVar(&flash, "f", false, "Flash extracted images to the device on current active slot(shorthand)")
	flag.BoolVar(&flash, "flash", false, "Flash extracted images to the device on current active slot")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	filename := flag.Arg(0)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s\n", filename)
	}

	if flash && !checkDevicesInFastbootMode() {
		fmt.Printf("Try with out -f or --flash\n")
		return
	}

	payloadBin := filename
	if strings.HasSuffix(filename, ".zip") {
		fmt.Println("Please wait while extracting payload.bin from the archive.")
		payloadBin = extractPayloadBin(filename)
		if payloadBin == "" {
			log.Fatal("Failed to extract payload.bin from the archive.")
		} else {
			defer os.Remove(payloadBin)
		}
	}
	fmt.Printf("payload.bin: %s\n", payloadBin)

	payload := NewPayload(payloadBin)
	if err := payload.Open(); err != nil {
		log.Fatal(err)
	}
	payload.Init()

	if list {
		return
	}

	if extractRecovery {
		partitions += "boot,dtbo,vendor_boot"
	}

	targetDirectory := outputDirectory
	if targetDirectory == "" {
		targetDirectory = getDirName(filename)
	}
	if _, err := os.Stat(targetDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(targetDirectory, 0o755); err != nil {
			log.Fatal("Failed to create target directory")
		}
	}
	fmt.Printf("Output Directory: %s\n", targetDirectory)

	payload.SetConcurrency(concurrency)
	fmt.Printf("Number of workers: %d\n", payload.GetConcurrency())

	if partitions != "" {
		if err := payload.ExtractSelected(targetDirectory, strings.Split(partitions, ",")); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := payload.ExtractAll(targetDirectory); err != nil {
			log.Fatal(err)
		}
	}

	if flash && flashHandler(targetDirectory) {
		fmt.Printf("Flashing completed successfully\n")
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [inputfile]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func getDirName(filename string) string {
	now := time.Now().Unix()

	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if ext == ".zip" {

		parts := strings.FieldsFunc(name, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '.'
		})

		if len(parts) >= 2 {
			return fmt.Sprintf("%s_%s_%d", parts[0], parts[1], now)
		}

		if len(parts) == 1 && parts[0] != "" {
			return fmt.Sprintf("%s_%d", parts[0], now)
		}
	}
	return fmt.Sprintf("extracted_%d", now)
}
