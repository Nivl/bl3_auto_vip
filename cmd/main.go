package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	bl3 "github.com/Nivl/blcodes"
	"github.com/shibukawa/configdir"
)

type flags struct {
	Email    string
	Password string
}

type config struct {
	cacheFileName string
	client        bl3.Bl3Client
	printer       bl3.Printer
}

func main() {
	f := &flags{}
	flag.StringVar(&f.Email, "email", "", "Email")
	flag.StringVar(&f.Password, "password", "", "Password")
	flag.Parse()

	cfg := &config{
		client:  bl3.NewBl3Client(),
		printer: bl3.NewPrinter(),
	}

	if err := run(f, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println("all done")
}

func run(f *flags, cfg *config) error {
	// validate user provided data
	f.Email = strings.TrimSpace(f.Email)
	if f.Email == "" {
		return errors.New("missing required flag: email")
	}
	f.Password = strings.TrimSpace(f.Password)
	if f.Password == "" {
		return errors.New("missing required flag: password")
	}

	// Set the default config options
	if cfg.cacheFileName == "" {
		cfg.cacheFileName = fmt.Sprintf("%x-shift-codes.json", md5.Sum([]byte(f.Email)))
	}

	// Log the user in
	cfg.printer.Start("Logging in as " + f.Email)
	if err := cfg.client.Login(f.Email, f.Password); err != nil {
		return fmt.Errorf("could not login: %w", err)
	}
	cfg.printer.Success()

	return redeemShift(cfg)
}

func redeemShift(cfg *config) error {
	cfg.printer.Start("Getting SHIFT platform list for your user")
	ownedPlatforms, err := cfg.client.GetUserPlatforms()
	if err != nil {
		return fmt.Errorf("could not get user's platforms: %w", err)
	}
	cfg.printer.Success()

	// TODO(melvin): Move to own function to remove all the elses
	cfg.printer.Start("Getting previously redeemed SHIFT codes")
	configDirs := configdir.New("bl3-auto-vip", "bl3-auto-vip")
	redeemedCodes := map[string]map[string]struct{}{}
	folder := configDirs.QueryFolderContainsFile(cfg.cacheFileName)
	if folder != nil {
		data, err := folder.ReadFile(cfg.cacheFileName)
		if err != nil {
			return fmt.Errorf("could not read cache file content: %w", err)
		}
		if err = json.Unmarshal(data, &redeemedCodes); err != nil {
			cfg.printer.Failed(fmt.Errorf("could not parse cache file content: %w", err).Error())
		} else {
			cfg.printer.Success()
		}
	} else {
		cfg.printer.Success()
	}

	cfg.printer.Start("Getting latest SHIFT codes")
	shiftCodes, err := cfg.client.GetFullShiftCodeList()
	if err != nil {
		return fmt.Errorf("could not get new SHIFT codes: %w", err)
	}
	cfg.printer.Success()

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

			// Let's redeem the code!
			foundCodes = true
			if _, ok := redeemedCodes[code.Code]; !ok {
				redeemedCodes[code.Code] = map[string]struct{}{}
			}
			redeemedCodes[code.Code][platform] = struct{}{}

			cfg.printer.Start(fmt.Sprintf(`Trying "%s" SHIFT code "%s"`, platform, code.Code))
			if err := cfg.client.RedeemShiftCode(code.Code, platform); err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "already") {
					cfg.printer.SuccessMsg("Already redeemed")
					continue
				}
				cfg.printer.Failed(fmt.Sprintf("Could not redeem: %s", err.Error()))
				delete(redeemedCodes, code.Code)
				continue

			}
			cfg.printer.SuccessMsg("Redeemed!")
		}
	}

	if !foundCodes {
		cfg.printer.Info("No new SHIFT codes at this time. Try again later.")
		return nil
	}

	folders := configDirs.QueryFolders(configdir.Global)
	data, err := json.Marshal(&redeemedCodes)
	if err != nil {
		return fmt.Errorf("could not JSON encode list of redeemed SHIFT codes: %w", err)
	}
	err = folders[0].WriteFile(cfg.cacheFileName, data)
	if err != nil {
		return fmt.Errorf("could not backup list of redeemed SHIFT codes: %w", err)
	}
	return nil
}
