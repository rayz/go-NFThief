package main

import (
	"fmt"
	"github.com/manifoldco/promptui"
	"go-NFThief/download"
	"os"
)

func main() {
	for {
		prompt := promptui.Select{
			Label: "Select Action",
			Items: []string{"Download a collection", "Download collection of owner", "Exit"},
		}

		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		switch result {
		case "Download a collection":
			download.DownloadByCollection()
		case "Download collection of owner":
			download.DownloadByOwner()
		case "Exit":
			os.Exit(0)
		}
	}

}
