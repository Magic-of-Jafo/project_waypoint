package main

import (
	"flag"
	"log"
	"os"

	"waypoint_archive_scripts/pkg/config"
)

func main() {
	// TODO: This will eventually be the main entry point for the Archiver.
	// For Story 2.5, we are focusing on configuration loading and logging politeness settings.

	// config.LoadConfig now expects arguments to parse.
	// It creates its own FlagSet. If parsing fails (e.g. -help or bad flag),
	// LoadConfig returns an error. We can choose to print usage here.
	cfg, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		// Check if the error is due to help flag
		// The flag package itself handles printing help message if flag.ExitOnError is used.
		// If flag.ContinueOnError is used in LoadConfig's FlagSet, Parse returns flag.ErrHelp.
		if err == flag.ErrHelp {
			// The FlagSet inside LoadConfig would have printed help if configured with ExitOnError.
			// If it was ContinueOnError, we might need to print usage here if desired,
			// but LoadConfig itself doesn't expose its internal FlagSet for us to call Usage().
			// For now, a general error message is logged by LoadConfig, and we exit.
			// A more sophisticated main might print its own top-level help.
			log.Printf("Displaying help due to -help flag or parsing error.")
			os.Exit(0) // Exit cleanly after help
		}
		// For other errors, LoadConfig already logs a warning.
		// We'll exit fatally here as configuration is critical.
		log.Fatalf("Critical error loading configuration: %v. Exiting.", err)
		// os.Exit(1) // log.Fatalf will exit
	}

	// AC5: Log active politeness settings at script startup
	log.Printf("[INFO] Archiver starting with Politeness Delay: %s", cfg.PolitenessDelay.String())
	log.Printf("[INFO] Archiver starting with User-Agent: %s", cfg.UserAgent)
	log.Printf("[INFO] Archiver using Archive Root Directory: %s", cfg.ArchiveRootDir)

	log.Println("Archiver initialized. (Further implementation pending)")
	// Future work:
	// Initialize Downloader with cfg
	// Initialize Storer with cfg
	// Loop through topics/pages to download and store
}
