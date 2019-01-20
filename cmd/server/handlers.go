package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rking788/destiny-gear-vendor/bungie"
)

func GetAsset(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	hash, ok := params["hash"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Forgot to specify an item hash"))
		return
	}
	format, ok := params["format"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Forgot to specify an asset format"))
		return
	}
	if format != "dae" && format != "stl" && format != "usd" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid asset format specified"))
		return
	}

	tempHash, err := strconv.ParseInt(hash, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid item hash provided"))
		return
	}

	fmt.Printf("Requesting item(%d) in format(%s)\n", tempHash, format)

	assetDefinition, err := bungie.GetAssetDefinition(uint(tempHash))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No item found with the specified item hash"))
		return
	}

	path := processGeometry(assetDefinition, (format == "stl"), (format == "dae"), (format == "usd"))
	if path == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong generating the model"))
		return
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to read the model file from disk"))
		return
	}

	// This should be different if the format was specified as STL
	w.Header().Set("Content-type", "model/vnd.collada+xml")
	filename := fmt.Sprintf("%s.%s", hash, format)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	http.ServeContent(w, r, format, time.Now(), bytes.NewReader(contents))
}
