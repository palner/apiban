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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/coreos/go-iptables/iptables"
	"github.com/palner/apiban/clients/go/apiban"
)

// ApibanConfig is the structure for the JSON config file
type ApibanConfig struct {
	APIKEY  string `json:"APIKEY"`
	LKID    string `json:"LKID"`
	VERSION string `json:"VERSION"`
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
	defer os.Exit(0)
	// Open our Log
	logfile, err := os.OpenFile("/var/log/apiban-client.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Panic(err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		runtime.Goexit()
	}
	defer logfile.Close()

	log.SetOutput(logfile)
	log.Print("** Started APIBAN CLIENT")
	log.Print("Licensed under GPLv2. See LICENSE for details.")

	// Open our config file
	ConfigFile, err := os.Open("/usr/local/bin/apiban/config.json")
	if err != nil {
		log.Panic(err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		runtime.Goexit()
	}
	defer ConfigFile.Close()

	// get config values
	ConfigValues, _ := ioutil.ReadAll(ConfigFile)
	var apiconfig ApibanConfig
	if err := json.Unmarshal(ConfigValues, &apiconfig); err != nil {
		log.Fatalln("failed to parse config:", err)
	}

	// if no APIKEY, exit
	if len(apiconfig.APIKEY) == 0 {
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

	// Get existing chains from IPTABLES
	originaListChain, err := ipt.ListChains("filter")
	if err != nil {
		log.Panic(err)
	}

	// Search for INPUT in IPTABLES
	chain := "INPUT"
	if !contains(originaListChain, chain) {
		log.Print("IPTABLES doesn't contain the chain ", chain)
		runtime.Goexit()
	}

	// Search for FORWARD in IPTABLES
	chain = "FORWARD"
	if !contains(originaListChain, chain) {
		log.Print("IPTABLES doesn't contain the chain ", chain)
		runtime.Goexit()
	}

	// Search for APIBAN in IPTABLES
	chain = "APIBAN"
	if !contains(originaListChain, chain) {
		log.Print("IPTABLES doesn't contain APIBAN. Creating now...")

		// Add APIBAN chain
		err = ipt.ClearChain("filter", chain)
		if err != nil {
			log.Panic(err)
		}

		// Add APIBAN chain to INPUT
		err = ipt.Insert("filter", "INPUT", 1, "-j", chain)
		if err != nil {
			log.Panic(err)
		}

		// Add APIBAN chain to FORWARD
		err = ipt.Insert("filter", "FORWARD", 1, "-j", chain)
		if err != nil {
			log.Panic(err)
		}
	}

	// Get list of banned ip's from APIBAN.org
	res, err := apiban.Banned(apiconfig.APIKEY, apiconfig.LKID)
	if err != nil {
		log.Fatalln("failed to get banned list:", err)
	}
	log.Print("got response")

	if res.ID == apiconfig.LKID {
		log.Print("Great news... no new bans to add. Exiting...")
		os.Exit(0)
	}

	if len(res.IPs) == 0 {
		log.Print("No IP addresses detected. Exiting.")
		os.Exit(0)
	}

	for _, ip := range res.IPs {
		blockedip := ip + "/32"
		err = ipt.AppendUnique("filter", "APIBAN", "-s", blockedip, "-d", "0/0", "-j", "REJECT")
		if err != nil {
			log.Print("Adding rule failed. ", err.Error())
		} else {
			log.Print("Blocking ", blockedip)
		}
	}

	// Update the config with the updated LKID
	UpdateConfig := bytes.Replace(ConfigValues, []byte("\""+apiconfig.LKID+"\""), []byte("\""+res.ID+"\""), -1)
	if err = ioutil.WriteFile("/usr/local/bin/apiban/config.json", UpdateConfig, 0666); err != nil {
		log.Panic(err)
		runtime.Goexit()
	}

	log.Print("** Done. Exiting.")
	runtime.Goexit()
}
