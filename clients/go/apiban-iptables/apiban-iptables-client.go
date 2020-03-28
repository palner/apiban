/*
 * Copyright (C) 2020 Fred Posner (palner.com)
 *
 * This file is part of APIBAN.org.
 *
 * apiban-iptables-client is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version
 *
 * apiban-iptables-client is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
 *
 */

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"github.com/palner/apiban/clients/go/apiban"
)

var configFileLocation string
var logFile string

func init() {
	flag.StringVar(&configFileLocation, "config", "", "location of configuration file")
	flag.StringVar(&logFile, "log", "/var/log/apiban-client.log", "location of log file or - for stdout")
}

// ApibanConfig is the structure for the JSON config file
type ApibanConfig struct {
	APIKEY  string `json:"APIKEY"`
	LKID    string `json:"LKID"`
	VERSION string `json:"VERSION"`

	sourceFile string
}

// Function to see if string within string
func contains(list []string, value string) bool {
	for _, val := range list {
		if val == value {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()

	defer os.Exit(0)

	// Open our Log
	if logFile != "-" && logFile != "stdout" {
		lf, err := os.OpenFile("/var/log/apiban-client.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Panic(err)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			runtime.Goexit()
		}
		defer lf.Close()

		log.SetOutput(lf)
	}

	log.Print("** Started APIBAN CLIENT")
	log.Print("Licensed under GPLv2. See LICENSE for details.")

	// Open our config file
	apiconfig, err := LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	// if no APIKEY, exit
	if apiconfig.APIKEY == "" {
		log.Fatalln("Invalid APIKEY. Exiting.")
	}

	// allow cli of FULL to reset LKID to 100
	if len(os.Args) > 1 {
		arg1 := os.Args[1]
		if arg1 == "FULL" {
			log.Print("CLI of FULL received, resetting LKID")
			apiconfig.LKID = "100"
		}
	} else {
		log.Print("no command line arguments received")
	}

	// if no LKID, reset it to 100
	if len(apiconfig.LKID) == 0 {
		log.Print("Resetting LKID")
		apiconfig.LKID = "100"
	}

	// Go connect for IPTABLES
	ipt, err := iptables.New()
	if err != nil {
		log.Panic(err)
	}

	if err := initializeIPTables(ipt); err != nil {
		log.Fatalln("failed to initialize IPTables:", err)
	}

	lastTimestamp, err := strconv.ParseInt(apiconfig.LKID, 10, 64)
	if err != nil {
		// If we don't have a valid timestamp, try 1 year ago
		lastTimestamp = time.Now().AddDate(-1, 0, 0).Unix()
	}

	// Get list of banned ip's from APIBAN.org
	list, err := apiban.NewOfficialStore(apiconfig.APIKEY).ListFromTime(time.Unix(lastTimestamp, 0))
	if err != nil {
		log.Fatalln("failed to get banned list:", err)
	}

	if len(list) == 0 {
		log.Print("Great news... no new bans to add. Exiting...")
		os.Exit(0)
	}

	for _, l := range list {
		blockedip := l.IP.String()

		err = ipt.AppendUnique("filter", "APIBAN", "-s", blockedip, "-d", "0/0", "-j", "REJECT")
		if err != nil {
			log.Print("Adding rule failed. ", err.Error())
		} else {
			log.Print("Blocking ", blockedip)
		}
	}

	// Update the config with the updated LKID
	apiconfig.LKID = strconv.FormatInt(list[len(list)-1].Timestamp.Unix(), 10)
	if err := apiconfig.Update(); err != nil {
		log.Fatalln(err)
	}

	log.Print("** Done. Exiting.")
}

// LoadConfig attempts to load the APIBAN configuration file from various locations
func LoadConfig() (*ApibanConfig, error) {
	var fileLocations []string

	// If we have a user-specified configuration file, use it preferentially
	if configFileLocation != "" {
		fileLocations = append(fileLocations, configFileLocation)
	}

	// If we can determine the user configuration directory, try there
	configDir, err := os.UserConfigDir()
	if err == nil {
		fileLocations = append(fileLocations, fmt.Sprintf("%s/apiban/config.json", configDir))
	}

	// Add standard static locations
	fileLocations = append(fileLocations,
		"/etc/apiban/config.json",
		"config.json",
		"/usr/local/bin/apiban/config.json",
	)

	for _, loc := range fileLocations {
		f, err := os.Open(loc)
		if err != nil {
			continue
		}
		defer f.Close()

		cfg := new(ApibanConfig)
		if err := json.NewDecoder(f).Decode(cfg); err != nil {
			return nil, fmt.Errorf("failed to read configuration from %s: %w", loc, err)
		}

		// Store the location of the config file so that we can update it later
		cfg.sourceFile = loc

		return cfg, nil
	}

	return nil, errors.New("failed to locate configuration file")
}

// Update rewrite the configuration file with and updated state (such as the LKID)
func (cfg *ApibanConfig) Update() error {
	f, err := os.Create(cfg.sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open configuration file for writing: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(cfg)
}

func initializeIPTables(ipt *iptables.IPTables) error {
	// Get existing chains from IPTABLES
	originaListChain, err := ipt.ListChains("filter")
	if err != nil {
		return fmt.Errorf("failed to read iptables: %w", err)
	}

	// Search for INPUT in IPTABLES
	chain := "INPUT"
	if !contains(originaListChain, chain) {
		return errors.New("iptables does not contain expected INPUT chain")
	}

	// Search for FORWARD in IPTABLES
	chain = "FORWARD"
	if !contains(originaListChain, chain) {
		return errors.New("iptables does not contain expected FORWARD chain")
	}

	// Search for APIBAN in IPTABLES
	chain = "APIBAN"
	if contains(originaListChain, chain) {
		// APIBAN chain already exists
		return nil
	}

	log.Print("IPTABLES doesn't contain APIBAN. Creating now...")

	// Add APIBAN chain
	err = ipt.ClearChain("filter", chain)
	if err != nil {
		return fmt.Errorf("failed to clear APIBAN chain: %w", err)
	}

	// Add APIBAN chain to INPUT
	err = ipt.Insert("filter", "INPUT", 1, "-j", chain)
	if err != nil {
		return fmt.Errorf("failed to add APIBAN chain to INPUT chain: %w", err)
	}

	// Add APIBAN chain to FORWARD
	err = ipt.Insert("filter", "FORWARD", 1, "-j", chain)
	if err != nil {
		return fmt.Errorf("failed to add APIBAN chain to FORWARD chain: %w", err)
	}

	return nil
}
