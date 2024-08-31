package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type SiteConfigNavElement struct {
	Title string `json:"title"`
	Href  string `json:"href"`
}

type SiteConfig struct {
	Name        string                 `json:"name"`
	Theme       string                 `json:"theme"`
	NavElements []SiteConfigNavElement `json:"nav-elements"`
}

type MetadataEntry struct {
	key   string
	value string
}

func ReadConfig() SiteConfig {
	rootPath := "./test-site"

	siteConfigFile, err := os.Open(fmt.Sprintf("%s/site.json", rootPath))
	if err != nil {
		fmt.Println("ERROR: Failed to open site.json")
	}
	defer siteConfigFile.Close()

	// unmarshal json and then return
	siteConfigData, _ := io.ReadAll(siteConfigFile)
	var siteConfig SiteConfig

	err = json.Unmarshal(siteConfigData, &siteConfig)
	if err != nil {
		fmt.Println("ERROR: Failed to parse site.json")
	}

	return siteConfig
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}
