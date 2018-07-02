package main

import (
	"errors"
	"flag"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"strings"

	"github.com/kpango/glg"
	"github.com/rking788/destiny-gear-vendor/graphics"
)

func main() {
	inPath := flag.String("path", "", "The path to the file that should be separated out into the individual physically based rendering components.")
	namePrefix := flag.String("prefix", "", "The prefix of the output files that will be written")

	flag.Parse()

	if *inPath == "" || *namePrefix == "" {
		glg.Errorf("Forgot to specify a path to the gearstack file or a name prefix")
		return
	}

	pbr, err := graphics.ExplodeGearstack(*inPath)
	if err != nil {
		glg.Errorf("Error expanding gearstack file: %s", err.Error())
		return
	}

	format := "jpeg"
	if strings.HasSuffix(*inPath, "png") {
		format = "png"
	}
	writeTextures(pbr, format, *namePrefix)
}

func writeTextures(pbr *graphics.PBRTextureCollection, format, prefix string) error {

	metalF, err := os.OpenFile(prefix+"_metalness."+format, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer metalF.Close()
	roughnessF, err := os.OpenFile(prefix+"_roughness."+format, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer roughnessF.Close()
	aoF, err := os.OpenFile(prefix+"_ao."+format, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer aoF.Close()

	if format == "png" {
		png.Encode(metalF, pbr.Metalness)
		png.Encode(roughnessF, pbr.Roughness)
		png.Encode(aoF, pbr.AmbientOcclusion)
	} else if format == "jpeg" {
		jpeg.Encode(metalF, pbr.Metalness, nil)
		jpeg.Encode(roughnessF, pbr.Roughness, nil)
		jpeg.Encode(aoF, pbr.AmbientOcclusion, nil)
	} else {
		return errors.New("Unknown image format " + format)
	}

	return nil
}
