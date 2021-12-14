package main

import (
	"fmt"
	"go-NFThief/download"
	"os"

	"github.com/manifoldco/promptui"
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
