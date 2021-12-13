package download

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
)

type asset struct {
	ImageUrl string `json:"image_url"`
	Name     string `json:"name"`
}

type openSeaAssetResponse struct {
	Assets []asset `json:"assets"`
}

type address struct {
	ContractAddress string `json:"address"`
}

type collection struct {
	PrimaryAssetContracts []address `json:"primary_asset_contracts"`
	Slug                  string    `json:"slug"`
}

func downloadAssets(slug string, assets []asset) {
	if _, err := os.Stat(slug); os.IsNotExist(err) {
		err := os.MkdirAll(slug, 0755)
		if err != nil {
			return
		}
		for _, asset := range assets {
			res, err := http.Get(asset.ImageUrl)
			if err != nil {
				fmt.Println("Could not download:", asset.ImageUrl)
				continue
			}
			defer res.Body.Close()
			h := fnv.New64a()
			h.Write([]byte(asset.ImageUrl))
			name := fmt.Sprint(h.Sum64())
			f := fmt.Sprintf("%s/%s", slug, name)
			fmt.Println("Downloading to:", f)
			out, err := os.Create(f)
			if err != nil {
				fmt.Println("Could not create:", f)
				continue
			}
			defer out.Close()
			_, err = io.Copy(out, res.Body)
		}
	} else {
		fmt.Println(slug, "folder already exists")
	}
}

func getAssets(slug, url string) []asset {

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	osap := openSeaAssetResponse{}
	err = json.Unmarshal(body, &osap)

	if err != nil {
		log.Fatal(err)
	}
	return osap.Assets
}

func getCollections(walletAddress string) []collection {
	url := fmt.Sprintf("https://api.opensea.io/api/v1/collections?asset_owner=%s&offset=0&limit=300", walletAddress)
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

	return collections
}

func DownloadByCollection() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter a collection name: ")
	ok := scanner.Scan()
	if ok != true {
		return
	}
	collectionName := strings.Join(strings.Split(strings.ToLower(scanner.Text()), " "), "")
	fmt.Println("Trying to download:", collectionName)
	url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?order_by=sale_price&order_direction=desc&offset=0&collection=%s&limit=50&", collectionName)
	assets := getAssets(collectionName, url)
	if len(assets) > 0 {
		downloadAssets(collectionName, assets)
	} else {
		fmt.Println("Collection", collectionName, "does not exist")
	}

}

func DownloadByOwner() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter owner's wallet address: ")
	ok := scanner.Scan()
	if ok != true {
		return
	}
	walletAddress := scanner.Text()
	collections := getCollections(walletAddress)
	for _, collection := range collections {
		if len(collection.PrimaryAssetContracts) > 0 {
			url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?owner=%s&asset_contract_address=%s&order_direction=desc&offset=0&limit=50", walletAddress, collection.PrimaryAssetContracts[0].ContractAddress)
			assets := getAssets(collection.Slug, url)
			slug := fmt.Sprintf("%s/%s", walletAddress, collection.Slug)
			downloadAssets(slug, assets)
		}
	}
}
