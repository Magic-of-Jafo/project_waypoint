module project-waypoint

go 1.23.0

toolchain go1.24.3

require (
	github.com/PuerkitoBio/goquery v1.10.3
	waypoint_archive_scripts v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/cascadia v1.3.3 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace waypoint_archive_scripts => ./waypoint_archive_scripts
