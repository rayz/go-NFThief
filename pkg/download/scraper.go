package download

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
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

type collectionStats struct {
	Stats struct {
		TotalSupply float64 `json:"total_supply"`
	} `json:"stats"`
}

func downloadAssets(slug string, assets []asset) {
	if _, err := os.Stat(slug); os.IsNotExist(err) {
		err := os.MkdirAll(slug, 0755)
		if err != nil {
			return
		}
		fmt.Println(slug, "folder created")
	}
	for _, asset := range assets {
		res, err := http.Get(asset.ImageUrl)
		if err != nil {
			fmt.Println("Could not download:", asset.ImageUrl)
			continue
		}
		if res.StatusCode == 429 { // slow down
			fmt.Println("Cooling down.. waiting 30 seconds")
			time.Sleep(time.Second * 30)
		}
		defer res.Body.Close()
		h := fnv.New64a()
		_, err = h.Write([]byte(asset.ImageUrl))
		if err != nil {
			fmt.Println("Could not hash:", asset.ImageUrl)
			continue
		}

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

		if err != nil {
			fmt.Println("Could not copy to:", f)
			continue
		}
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

func getStats(slug string) collectionStats {
	url := fmt.Sprintf("https://api.opensea.io/api/v1/collection/%s/stats", slug)
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	stat := collectionStats{}
	err = json.Unmarshal(body, &stat)

	if err != nil {
		log.Fatal(err)
	}
	return stat
}

func DownloadByCollection() {
	start := time.Now()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter a collection name: ")
	ok := scanner.Scan()
	if !ok {
		return
	}
	collectionName := strings.Join(strings.Split(strings.ToLower(scanner.Text()), " "), "")
	fmt.Println("Trying to download:", collectionName)

	stats := getStats(collectionName)
	supply := stats.Stats.TotalSupply

	var wg sync.WaitGroup
	for i := 0.0; i < math.Ceil(supply/50); i++ {
		url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?order_direction=desc&offset=%d&collection=%s&limit=50&", int(i*50), collectionName)
		assets := getAssets(collectionName, url)
		if len(assets) > 0 {
			wg.Add(1)
			go func(a []asset) {
				downloadAssets(collectionName, a)
				wg.Done()
			}(assets)
		} else {
			fmt.Println("Could not download assets from", url)
		}
	}
	wg.Wait()
	fmt.Println(time.Since(start), "taken to download", collectionName, "collection")
}

func DownloadByOwner() {
	start := time.Now()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter owner's wallet address: ")
	ok := scanner.Scan()
	if !ok {
		return
	}
	walletAddress := scanner.Text()
	collections := getCollections(walletAddress)
	var wg sync.WaitGroup
	for _, collection := range collections {
		if len(collection.PrimaryAssetContracts) > 0 {
			url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?owner=%s&asset_contract_address=%s&order_direction=desc&offset=0&limit=50", walletAddress, collection.PrimaryAssetContracts[0].ContractAddress)
			assets := getAssets(collection.Slug, url)
			if len(assets) > 0 {
				wg.Add(1)
				go func(a []asset) {
					slug := fmt.Sprintf("%s/%s", walletAddress, collection.Slug)
					downloadAssets(slug, assets)
					wg.Done()
				}(assets)

			}
		}
	}
	wg.Wait()
	fmt.Println(time.Since(start), "taken to download collection of:", walletAddress)
}
