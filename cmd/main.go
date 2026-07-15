package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/leaanthony/spinner"
	"github.com/manifoldco/promptui"
	t "github.com/mzdrale/tssh/teleport"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var config Config

var (
	binName string
	version string
	cfgDir  string
	cfgFile string
)

// Argument variables
var (
	aPrintVersion bool
	aUpdateNodes  bool
)

func init() {
	userHomeDir, err := t.GetLocalUserHomeDir()
	if err != nil {
		log.Printf("[ERROR] Couldn't determine your homedir: %s\n\n", err)
	}

	cfgDir = path.Join(userHomeDir, ".config/tssh")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.AddConfigPath(cfgDir)
	}

	// Try to read config
	if err := viper.ReadInConfig(); err != nil {
		fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Unable to read configuration file: %s\n\n", err.Error())))
		os.Exit(1)
	}

	// Usage
	flag.Usage = func() {
		fmt.Printf("Usage: \n")
		flag.PrintDefaults()
	}

	// Get arguments
	flag.BoolVarP(&aPrintVersion, "version", "V", false, "Print version")
	flag.BoolVarP(&aUpdateNodes, "update-nodes", "u", false, "Update nodes from Teleport")
	flag.Parse()

}

func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func searchAndSelectNode(nodes []t.Node, config *Config) (*t.Node, error) {
	// Map color names to functions
	colorFuncs := map[string]func(...interface{}) string{
		"Yellow":  t.Yellow,
		"Teal":    t.Teal,
		"Magenta": t.Magenta,
		"Red":     t.Red,
		"Green":   t.Green,
		"White":   t.White,
	}

	// Prepare items for promptui
	items := make([]string, len(nodes))
	for i, node := range nodes {
		var envColor func(...interface{}) string

		// Get color from config
		if envConfig, exists := config.Environments[node.Env]; exists {
			if colorFunc, found := colorFuncs[envConfig.Color]; found {
				envColor = colorFunc
			} else {
				envColor = t.White // fallback
			}
		} else {
			envColor = t.White // fallback
		}

		items[i] = fmt.Sprintf("%s [%s]", t.White(node.Hostname), envColor(node.Env))
	}

	prompt := promptui.Select{
		Label: "Search and select node by hostname",
		Items: items,
		Searcher: func(input string, index int) bool {
			node := nodes[index]
			name := strings.ToLower(node.Hostname)
			input = strings.ToLower(input)
			return strings.Contains(name, input)
		},
		Size: 30,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return &nodes[i], nil
}

func main() {
	// Clear the terminal screen
	fmt.Print("\033[H\033[2J")

	if aPrintVersion {
		fmt.Printf("\n%v %v\n\n", binName, version)
		fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
		fmt.Printf("URL: https://github.com/mzdrale/tssh\n\n")
		os.Exit(0)
	}

	config, err := LoadConfig(viper.ConfigFileUsed())
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	nodesFile := path.Join(cfgDir, "nodes.json")
	profileName := config.Default.Profile
	proxy := config.Profiles[profileName].Proxy
	username := config.Profiles[profileName].Username
	cacheValidity := config.Default.CacheValidity
	if cacheValidity == "" {
		cacheValidity = "24" // Default to 24 hours if not set
	}

	// Convert string to duration
	cacheValidityDuration, err := time.ParseDuration(cacheValidity + "h")
	if err != nil {
		fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Invalid cache validity duration: %s\n\n", err.Error())))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(t.KeyText("Profile:"), profileName)
	fmt.Println(t.KeyText("Proxy:"), proxy)
	fmt.Println(t.KeyText("Username:"), username)
	fmt.Println()

	// Check if hostname was provided as argument
	args := flag.Args()
	var selectedNode *t.Node

	if len(args) > 0 {
		targetHostname := args[0]
		// Load nodes to find the matching hostname
		nodes, err := t.LoadNodesFromFile(nodesFile)
		if err != nil {
			fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Unable to load nodes from file: %s\n\n", err.Error())))
			os.Exit(1)
		}

		// Find node by hostname
		for i, node := range nodes {
			if node.Hostname == targetHostname {
				selectedNode = &nodes[i]
				break
			}
		}

		if selectedNode == nil {
			// If no exact match found, search for nodes containing the search term
			var filteredNodes []t.Node
			searchTerm := strings.ToLower(targetHostname)

			for _, node := range nodes {
				if strings.Contains(strings.ToLower(node.Hostname), searchTerm) {
					filteredNodes = append(filteredNodes, node)
				}
			}

			if len(filteredNodes) == 0 {
				fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 No nodes found matching '%s'\n\n", targetHostname)))
				os.Exit(1)
			}

			// Sort filtered nodes by hostname
			sort.Slice(filteredNodes, func(i, j int) bool {
				return filteredNodes[i].Hostname < filteredNodes[j].Hostname
			})

			// Let user select from filtered nodes
			selectedNode, err = searchAndSelectNode(filteredNodes, config)
			if err != nil {
				// If the error is due to user pressing ctrl+c (SIGINT), exit normally with code 0
				if err == promptui.ErrInterrupt {
					fmt.Printf("\n\U0001F44B Bye!\n")
					os.Exit(0)
				}
				fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Prompt failed: %s\n\n", err)))
				os.Exit(1)
			}
		}
	} else {
		// If nodesFile is older than 1 day, update nodes
		outdatedList := false
		if !aUpdateNodes {
			if _, err := os.Stat(nodesFile); err == nil {
				fileInfo, err := os.Stat(nodesFile)
				if err != nil {
					fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Unable to get file info: %s\n\n", err.Error())))
					os.Exit(1)
				}

				// If nodesFile is older than 1 day, update nodes
				if time.Since(fileInfo.ModTime()) > cacheValidityDuration {
					fmt.Print(t.Warn("\U00002757Nodes file is older than 1 day!\n"))
					outdatedList = true
				}
			}
		}

		var nodes []t.Node
		if aUpdateNodes || outdatedList {
			// Create a new spinner
			s := spinner.New(t.Info("Updating nodes list from Teleport"))
			s.SetAbortMessage("Aborted")
			s.Start() // Start the spinner

			nodes, err = t.GetNodes(proxy)
			if err != nil {
				fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Unable to get nodes list from Teleport: %s\n\n", err.Error())))
				os.Exit(1)
			}

			if err := t.SaveNodesToFile(nodes, nodesFile); err != nil {
				fmt.Print(t.Fatal(fmt.Sprintf("Unable to save nodes to file: %s\n\n", err.Error())))
				os.Exit(1)
			} else {
				s.Success(t.Info("Updated nodes list from Teleport\n")) // Stop the spinner
			}

			if aUpdateNodes {
				os.Exit(1)
			}
		}

		nodes, err = t.LoadNodesFromFile(nodesFile)
		if err != nil {
			fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Unable to load nodes from file: %s\n\n", err.Error())))
			os.Exit(1)
		}

		// Sort nodes by hostname
		if err == nil {
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Hostname < nodes[j].Hostname
			})
		}

		if err != nil {
			fmt.Print(t.Fatal("\U00002717 Unable to get nodes: %s\n\n", err.Error()))
			os.Exit(1)
		}

		selectedNode, err = searchAndSelectNode(nodes, config)
		if err != nil {
			// If the error is due to user pressing ctrl+c (SIGINT), exit normally with code 0
			if err == promptui.ErrInterrupt {
				fmt.Printf("\n\U0001F44B Bye!\n")
				os.Exit(0)
			}
			fmt.Println(t.Fatal("\U00002717 Prompt failed: %s\n\n", err))
			os.Exit(1)
		}

	}

	fmt.Print(t.Grey("Selected node: \n\n"))
	fmt.Printf(t.KeyText("   Hostname: %s\n"), t.Yellow(selectedNode.Hostname))
	fmt.Printf(t.KeyText("   UUID: %s\n"), t.White(selectedNode.UUID))
	fmt.Printf(t.KeyText("   Env: %s\n"), t.White(selectedNode.Env))
	fmt.Printf(t.KeyText("   Teleport Version: %s\n"), t.White(selectedNode.Version))
	if selectedNode.CloudMetadata != nil && selectedNode.CloudMetadata.AWS != nil {
		aws := selectedNode.CloudMetadata.AWS
		fmt.Printf(t.KeyText("   Cloud: %s\n"), t.White("AWS"))
		fmt.Printf(t.KeyText("   Account ID: %s\n"), t.White(aws.AccountID))
		fmt.Printf(t.KeyText("   Instance ID: %s\n"), t.White(aws.InstanceID))
	}
	fmt.Println()

	// Run tsh ssh node.Hostname
	cmd := exec.Command("tsh", "ssh", username+"@"+selectedNode.Hostname)
	fmt.Printf(t.Grey("   Running: %s\n\n"), t.White(cmd.String()))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Print(t.Fatal(fmt.Sprintf("\U00002717 Failed to run tsh ssh: %v\n", err)))
		os.Exit(1)
	}
}
