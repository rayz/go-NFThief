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
	"strconv"
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

func downloadAssets(slug, target string, assets []asset, wg *sync.WaitGroup, limits chan bool) {
	for _, asset := range assets {
		res, err := http.Get(asset.ImageUrl)
		if err != nil {
			fmt.Println("Could not download:", asset.ImageUrl, err)
			continue
		}
		h := fnv.New64a()
		_, err = h.Write([]byte(asset.ImageUrl))
		if err != nil {
			fmt.Println("Could not hash:", asset.ImageUrl)
			continue
		}

		name := fmt.Sprint(h.Sum64())
		f := fmt.Sprintf("%s/%s", target, name)
		//		fmt.Println("Downloading to:", f)
		out, err := os.Create(f)
		if err != nil {
			fmt.Println("Could not create:", f)
			continue
		}
		defer out.Close()
		_, err = io.Copy(out, res.Body)
		res.Body.Close()

		if err != nil {
			fmt.Println("Could not copy to:", f)
			continue
		}
	}

}

func getAssets(slug, url, target string, wg *sync.WaitGroup, limits chan bool) {
	limits <- true

	defer wg.Done()

	defer func() {
		<-limits
	}()

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode == 429 { // slow down
		retry_after := res.Header.Get("Retry-After")
		wait, _ := strconv.Atoi(retry_after)
		time.Sleep(time.Second * time.Duration(wait))
		res, err = http.Get(url)
	}

	body, _ := ioutil.ReadAll(res.Body)

	res.Body.Close()

	osap := openSeaAssetResponse{}
	err = json.Unmarshal(body, &osap)

	if err != nil {
		log.Fatal(err)
	}

	if len(osap.Assets) > 0 {
		downloadAssets(slug, target, osap.Assets, wg, limits)
	} else {
		fmt.Println("Could not download", url)
		fmt.Println(osap.Assets)
	}
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
	if _, err := os.Stat(collectionName); os.IsNotExist(err) {
		err := os.MkdirAll(collectionName, 0755)
		if err != nil {
			return
		}
		fmt.Println(collectionName, "folder created")
	}
	fmt.Println("Trying to download:", collectionName)

	stats := getStats(collectionName)
	supply := stats.Stats.TotalSupply

	wg := new(sync.WaitGroup)

	workers := 7
	limits := make(chan bool, workers)
	wg.Add(int(math.Ceil(supply / 50)))
	for i := 0.0; i < math.Ceil(supply/50); i++ {
		url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?order_direction=desc&offset=%d&collection=%s&limit=50", int(i*50), collectionName)
		go func() {
			getAssets(collectionName, url, collectionName, wg, limits)
		}()
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
	wg := new(sync.WaitGroup)

	workers := 7
	limits := make(chan bool, workers)
	for _, collection := range collections {
		if len(collection.PrimaryAssetContracts) > 0 {
			url := fmt.Sprintf("https://api.opensea.io/api/v1/assets?owner=%s&asset_contract_address=%s&order_direction=desc&offset=0&limit=50", walletAddress, collection.PrimaryAssetContracts[0].ContractAddress)
			target := fmt.Sprintf("%s/%s", walletAddress, collection.Slug)
			go func() {
				getAssets(collection.Slug, url, target, wg, limits)
			}()
		}
	}
	wg.Wait()
	fmt.Println(time.Since(start), "taken to download collection of:", walletAddress)
}
