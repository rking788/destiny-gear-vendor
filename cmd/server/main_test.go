package main

import (
	"archive/zip"
	"fmt"
	"testing"
)

func TestCreateUSDZ(t *testing.T) {

	_, err := createUSDZ("../../output/gear.scnassets/2069224589/", 2069224589)
	if err != nil {
		t.Errorf("Failed with error: %s", err.Error())
	}
}

func TestZipFileHeaders(t *testing.T) {

	unaligned := "../../output/gear.scnassets/2069224589/2069224589.usdz"
	aligned := "../../output/gear.scnassets/2069224589/2069224589-android-tools.usdz"

	// Unaligned
	zipReader, err := zip.OpenReader(unaligned)
	if err != nil {
		t.Errorf("Error opening unaligned: %s", err.Error())
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		offset, _ := f.DataOffset()
		fmt.Printf("File(%s), Offset=%d\nFileHeaderInfo:%+v\n", f.Name, offset, f.FileHeader)
	}

	fmt.Printf("\n\n ======= STARTING ALIGNED =======\n\n\n")

	// Aligned
	alignedZipReader, err := zip.OpenReader(aligned)
	if err != nil {
		t.Errorf("Error opening aligned: %s", err.Error())
	}
	defer alignedZipReader.Close()

	for _, f := range alignedZipReader.File {
		offset, _ := f.DataOffset()
		fmt.Printf("File(%s), Offset=%d\nFileHeaderInfo:%+v\n", f.Name, offset, f.FileHeader)
	}
}
