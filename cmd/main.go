package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	bl3 "github.com/matt1484/bl3_auto_vip"
	"github.com/shibukawa/configdir"
)

// gross but effective for now
const version = "2.1"

var usernameHash string

type flags struct {
	Username        string
	Password        string
	SingleShiftCode string
	AllowInactive   bool
}

func main() {
	f := &flags{}

	flag.StringVar(&f.Username, "email", "", "Email")
	flag.StringVar(&f.Password, "psw", "", "Password")
	flag.Parse()

	if err := run(f); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println("all done")
}

func run(f *flags) error {
	hasher := md5.New()
	_, err := hasher.Write([]byte(f.Username))
	if err != nil {
		return fmt.Errorf("could not hash username: %w", err)
	}
	usernameHash = hex.EncodeToString(hasher.Sum(nil))

	fmt.Print("Setting up...")
	client, err := bl3.NewBl3Client()
	if err != nil {
		return fmt.Errorf("could not create bl3 client: %w", err)
	}
	fmt.Println("success!")

	fmt.Printf(`Logging in as "%s"...`, f.Username)
	err = client.Login(f.Username, f.Password)
	if err != nil {
		return fmt.Errorf("could not login: %w", err)
	}
	fmt.Println("success!")

	if err := redeemShift(client); err != nil {
		return err
	}

	// return redeemVIP(client)
	return nil
}

func redeemShift(clt bl3.Bl3Client) error {
	fmt.Print("Getting SHIFT platform list for your user...")
	ownedPlatforms, err := clt.GetUserPlatforms()
	if err != nil {
		return fmt.Errorf("could not get user's platforms: %w", err)
	}
	fmt.Println("done!")

	fmt.Print("Getting previously redeemed SHIFT codes...")
	configDirs := configdir.New("bl3-auto-vip", "bl3-auto-vip")
	configFilename := usernameHash + "-shift-codes.json"
	redeemedCodes := map[string]map[string]struct{}{}
	folder := configDirs.QueryFolderContainsFile(configFilename)
	if folder != nil {
		data, err := folder.ReadFile(configFilename)
		if err == nil && data != nil {
			if json := bl3.JsonFromBytes(data); json != nil {
				json.Out(&redeemedCodes)
			}
		}
	}
	fmt.Println("done!")

	fmt.Print("Getting latest SHIFT codes...")
	shiftCodes, err := clt.GetFullShiftCodeList()
	if err != nil {
		return fmt.Errorf("could not get new SHIFT codes: %w", err)
	}
	fmt.Println("done!")

	foundCodes := false
	for _, code := range shiftCodes {
		// If the code is universal, we automatically set the platform
		// list to the user's (otherwise the list is empty)
		if code.IsUniversal {
			code.Platforms = ownedPlatforms
		}
		for platform := range code.Platforms {
			// We first check if the code is available on the user's
			// platforms
			if !code.IsUniversal {
				if _, owned := ownedPlatforms[platform]; !owned {
					continue
				}
			}
			// We now make sure the code hasn't been redeemed yet
			if _, codeRedeemed := redeemedCodes[code.Code]; codeRedeemed {
				if _, redeemed := redeemedCodes[code.Code][platform]; redeemed {
					continue
				}
			}

			// We redeem the code!
			foundCodes = true
			fmt.Printf(`Trying "%s" SHIFT code "%s"...`, platform, code.Code)
			if err := clt.RedeemShiftCode(code.Code, platform); err != nil {
				lcErr := strings.ToLower(err.Error())
				if !strings.Contains(lcErr, "already") {
					fmt.Printf("Could not redeem: %s\n", err.Error())
					continue
				}
				fmt.Println("Already redeemed")
			}
			if _, ok := redeemedCodes[code.Code]; !ok {
				redeemedCodes[code.Code] = map[string]struct{}{}
			}
			redeemedCodes[code.Code][platform] = struct{}{}
			fmt.Println("Redeemed!")
		}
	}

	if !foundCodes {
		fmt.Println("No new SHIFT codes at this time. Try again later.")
		return nil
	}

	folders := configDirs.QueryFolders(configdir.Global)
	data, err := json.Marshal(&redeemedCodes)
	if err != nil {
		return fmt.Errorf("could not JSON encode list of redeemed SHIFT codes: %w", err)
	}
	err = folders[0].WriteFile(configFilename, data)
	if err != nil {
		return fmt.Errorf("could not backup list of redeemed SHIFT codes: %w", err)
	}
	return nil
}

// func redeemVIP(client *bl3.Bl3Client) error {
// 	fmt.Print("Getting available VIP activities (excluding codes)...")
// 	activities, err := client.GetVipActivities()
// 	if err != nil {
// 		return fmt.Errorf("could not get VIP activities %w", err)
// 	}
// 	fmt.Println("success!")

// 	foundActivities := false
// 	for _, activity := range activities {
// 		title := strings.ToLower(activity.Title)
// 		link := strings.ToLower(activity.Link)
// 		if !strings.Contains(title, "watch") && !strings.Contains(link, "video") {
// 			fmt.Print(`Trying VIP activity "%s"...`, activity.Title)
// 			foundActivities = true
// 			status := "failed!"
// 			if client.RedeemVipActivity(activity) {
// 				status = "done!"
// 			}
// 			fmt.Println(status)
// 		}
// 	}
// 	if !foundActivities {
// 		fmt.Println("No new VIP activities at this time. Try again later.")
// 	}

// 	configDirs := configdir.New("bl3-auto-vip", "bl3-auto-vip")
// 	configFilename := usernameHash + "-vip-codes.json"
// 	redeemedCodesCached := bl3.VipCodeMap{}

// 	fmt.Print("Getting previously redeemed VIP codes...")
// 	folder := configDirs.QueryFolderContainsFile(configFilename)
// 	if folder != nil {
// 		data, err := folder.ReadFile(configFilename)
// 		if err == nil && data != nil {
// 			if json := bl3.JsonFromBytes(data); json != nil {
// 				json.Out(&redeemedCodesCached)
// 			}
// 		}
// 	}
// 	redeemedCodes, err := client.GetRedeemedVipCodeMap()
// 	if err != nil {
// 		return fmt.Errorf("%w", err)
// 	}
// 	for codeType, codes := range redeemedCodesCached {
// 		for code := range codes {
// 			redeemedCodes.Add(codeType, code)
// 		}
// 	}
// 	fmt.Println("done!")

// 	fmt.Print("Getting new VIP codes...")
// 	allCodes, err := client.GetFullVipCodeMap()
// 	if err != nil {
// 		return fmt.Errorf("could not getting new VIP codes: %w", err)
// 	}
// 	fmt.Println("done!")

// 	newCodes := allCodes.Diff(redeemedCodes)
// 	foundCodes := false
// 	for codeType, codes := range newCodes {
// 		if len(codes) < 1 {
// 			continue
// 		}
// 		foundCodes = true
// 		fmt.Printf(`Setting up VIP codes of type "%s"...`, codeType)
// 		_, found := client.Config.Vip.CodeTypeUrlMap[codeType]
// 		if !found {
// 			fmt.Println("invalid! Moving on.")
// 			continue
// 		}
// 		fmt.Println("success!")

// 		for code := range codes {
// 			fmt.Printf(`Trying "%s" VIP code "%s"...`, codeType, code)
// 			res, valid := client.RedeemVipCode(codeType, code)
// 			if !valid {
// 				fmt.Println("failed! Moving on.")
// 				continue
// 			}
// 			redeemedCodes.Add(codeType, code)
// 			fmt.Println(res)
// 		}
// 	}

// 	if !foundCodes {
// 		fmt.Println("No new VIP codes at this time. Try again later.")
// 		return nil
// 	}

// 	folders := configDirs.QueryFolders(configdir.Global)
// 	data, err := json.Marshal(&redeemedCodes)
// 	if err != nil {
// 		return fmt.Errorf("could not JSON encode list of redeemed VIP codes: %w", err)
// 	}
// 	err = folders[0].WriteFile(configFilename, data)
// 	if err != nil {
// 		return fmt.Errorf("could not backup list of redeemed VIP codes: %w", err)
// 	}

// 	return nil
// }
