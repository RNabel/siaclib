package siaclib

import (
	"net/http"
	"bytes"
	"log"
	"encoding/json"
	"time"
)

var (
	BASE_ADDRESS = "http://127.0.0.1:9980"
	RENTER = "/renter"
	CONTRACTS = RENTER + "/contracts"
	DELETE = RENTER + "/delete"
	DOWNLOADS = RENTER + "/downloads"
	DOWNLOAD = RENTER + "/download"
	FILES = RENTER + "/files"
	UPLOAD = RENTER + "/upload"
)

type FileCond func(File)bool

func Delete(rem string) (string, error) {
	return makeRequest(DELETE + "/" + rem, "POST", nil)
}

func Download(rem string, local string) (string, error) {
	// String map needed to pass URL parameters.
	var m map[string]string
	m = make(map[string]string)
	m["destination"] = local

	js, err := getWithArgs(DOWNLOAD + "/" + rem, m)
	return js, err
}

func Downloads() (DownloadList, error) {
	js, err := get(DOWNLOADS)

	var dat DownloadList

	decode(js, &dat)

	return dat, err
}

func ListFiles() (FileList, error) {
	js, err := get(FILES)
	var dat FileList

	decode(js, &dat)

	return dat, err
}

// Blocking upload call. Upload allows precise setting of replication 
// 	parameters, which are not currently supported.
func UploadDefault(src string, dst string) (error) {
	var m map[string]string
	m = make(map[string]string)
	m["source"] = src

	// Initialize upload.
	_, err := makeRequest(UPLOAD + "/" + dst, "POST", m)
	if err != nil {
		return err
	}

	// Wait until available.
	// 	Define the `available` condition, by filtering for the correct Siapath and Available
	// 	flag.
	avlbl := func(f File)bool {
		return f.Siapath == dst && f.Available
	}
	waitForCondition(avlbl)

	return nil
}

func WaitForRedundancy(fname string, redlevel float32) {
	// Define a redundancy condition.
	red := func(f File)bool {
		return f.Siapath == fname && f.Redundancy >= redlevel
	}
	waitForCondition(red)
}

// HELPERS.
// ========
func decode(js string, dat interface{}) {
	err := json.Unmarshal([]byte(js), &dat)
	if err != nil {
		log.Fatalln(err)
	}
}

func getWithArgs(endpoint string, args map[string]string) (string, error) {
	js, err := makeRequest(endpoint, "GET", args)

	return js, err
}

func get(endpoint string) (string, error) {
	return getWithArgs(endpoint, nil)
}

func makeRequest(endpoint string, requestType string, args map[string]string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest(requestType, BASE_ADDRESS + endpoint, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "Sia-Agent")

	// Add query parameters.
	q := req.URL.Query()
	for key, val := range args {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()

	// Execute request.
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	return buf.String(), nil
}

func waitForCondition(f FileCond) error {
	// Check for upload progress in regular intervals.
	cont := true
	for cont {
		dat, err := ListFiles()
		if err != nil {
			return err
		}

		for _, val := range dat.Files {
			if f(val) {
				cont = false
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}


// TYPE DEFINITIONS.
// =================
type FileList struct {
	Files []File`json:"files"`
}

type File struct {
	Siapath        string `json:"siapath"`
	Filesize       int `json:"filesize"`
	Available      bool `json:"available"`
	Renewing       bool `json:"renewing"`
	Redundancy     float32 `json:"redundancy"`
	Uploadprogress float32 `json:"uploadprogress"`
	Expiration     int `json:"expiration"`
}

type DownloadList struct {
	Downloads []struct {
		Siapath string `json:"siapath"`
		Destination string `json:"destination"`
		Filesize int `json:"filesize"`
		Received int `json:"received"`
		Starttime time.Time `json:"starttime"`
		Error string `json:"error"`
	} `json:"downloads"`
}
