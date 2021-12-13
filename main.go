package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

type asset struct {
	ImageUrl string `json:"image_url"`
	Name     string `json:"name"`
}

type OpenSeaAssetResponse struct {
	Assets []asset `json:"assets"`
}

type collection struct {
	Slug string `json:"slug"`
}

func downloadAssets(slug string, assets []asset) {
	if _, err := os.Stat(slug); os.IsNotExist(err) {
		err := os.Mkdir(slug, 0755)
		if err != nil {
			log.Fatal(err)
		}
		for _, asset := range assets {
			res, err := http.Get(asset.ImageUrl)
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			h := fnv.New64a()
			h.Write([]byte(asset.ImageUrl))
			name := fmt.Sprint(h.Sum64())
			f := fmt.Sprintf("%s/%s", slug, name)
			fmt.Println(f, "is the path!")
			out, err := os.Create(f)
			if err != nil {
				log.Fatal(err)
			}
			defer out.Close()
			_, err = io.Copy(out, res.Body)
		}
	} else {
		fmt.Println(slug, "folder already exists")
	}
}

func getAssets(slug string) []asset {
	url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?order_by=sale_price&order_direction=desc&offset=0&collection=%s&limit=50&", slug)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	osap := OpenSeaAssetResponse{}
	err = json.Unmarshal(body, &osap)

	if err != nil {
		log.Fatal(err)
	}
	return osap.Assets
}

func getCollections(walletAddress string) {
	url := "https://api.opensea.io/api/v1/collections?asset_owner=0x72e7212EF9d93244C93BF4DB64E69b582CcaC0D4&offset=0&limit=300"
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	collections := []collection{}

	err = json.Unmarshal(body, &collections)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
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
			fmt.Print("Enter a collection name: ")
			ok := scanner.Scan()
			if ok != true {
				break
			}
			collectionName := strings.Join(strings.Split(strings.ToLower(scanner.Text()), " "), "")
			fmt.Println("Trying to download:", collectionName)
			assets := getAssets(collectionName)
			if len(assets) > 0 {
				fmt.Println("Going to download..")
				downloadAssets(collectionName, assets)
			} else {
				fmt.Println("Collection", collectionName, "does not exist")
			}
		case "Download collection of owner":
			//			fmt.Print("Enter owner's wallet address: ")
			//			ok := scanner.Scan()
			//			if ok != true {
			//				break
			//			}
			//			walletAddress := scanner.Text()
			walletAddress := "0x72e7212EF9d93244C93BF4DB64E69b582CcaC0D4"
			getCollections(walletAddress)

		case "Exit":
			os.Exit(0)
		}
	}

}
