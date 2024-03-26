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
	selfupdate.EnableLog()
	currentVersion := semver.MustParse(VERSION)
	log.Infof("current version is %s", currentVersion)
	log.Info("Checking for update...")
	latest, found, err = selfupdate.DetectLatest("henrygd/webvids")

	if err != nil {
		log.Error(err.Error(), "err", err)
	}

	if !found {
		log.Info("found", found)
		log.Info("latest", latest)
		log.Error("No releases found")
		os.Exit(1)
	}

	if latest.Version.LTE(currentVersion) {
		log.Infof("Already up to date: %s", latest.Version)
		return
	}

	log.Infof("Found new version: %s", latest.Version)

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
	log.Infof("Successfully updated: %s -> %s\n\nRelease note:\n%s", VERSION, latest.Version, strings.TrimSpace(latest.ReleaseNotes))
}
