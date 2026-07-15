package main

type Config struct {
	Default struct {
		Profile       string `toml:"profile"`
		CacheValidity string `toml:"cache_validity"` // Duration in hours, e.g., "24" for 24 hours
	} `toml:"default"`
	Profiles map[string]struct {
		Proxy    string `toml:"proxy"`
		Username string `toml:"username"`
	} `toml:"profiles"`
	Environments map[string]struct {
		Color string `toml:"color"`
	} `toml:"environments"`
}
