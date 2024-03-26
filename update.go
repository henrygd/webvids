package main

import (
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/charmbracelet/log"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func Update() {
	var latest *selfupdate.Release
	var found bool
	var err error
	currentVersion := semver.MustParse(VERSION)
	log.Info("Checking for updates...")
	log.Infof("Current version is %s", currentVersion)
	latest, found, err = selfupdate.DetectLatest("henrygd/webvids")

	if err != nil {
		log.Error(err.Error(), "err", err)
	}

	if !found {
		log.Error("Could not find any releases")
		os.Exit(1)
	}

	log.Infof("Latest version is %s", latest.Version)

	if latest.Version.LTE(currentVersion) {
		log.Info("You are up to date")
		return
	}

	var binaryPath string
	log.Infof("Updating from %s to %s...", VERSION, latest.Version)
	binaryPath, err = os.Executable()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	err = selfupdate.UpdateTo(latest.AssetURL, binaryPath)
	if err != nil {
		log.Error("Please try running the command using sudo", "err", err)
		os.Exit(1)
	}
	log.Infof("Successfully updated: %s -> %s\n\n%s", VERSION, latest.Version, strings.TrimSpace(latest.ReleaseNotes))
}
