package teleport

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type AWSMetadata struct {
	AccountID  string `json:"account_id"`
	InstanceID string `json:"instance_id"`
}

type CloudMetadata struct {
	AWS *AWSMetadata `json:"aws,omitempty"`
}

type Node struct {
	UUID          string            `json:"uuid"`
	Labels        map[string]string `json:"labels"`
	Hostname      string            `json:"hostname"`
	NewHostname   string            `json:"new_hostname"`
	Addr          string            `json:"addr"`
	Version       string            `json:"version"`
	Env           string            `json:"env"`
	CloudMetadata *CloudMetadata    `json:"cloud_metadata,omitempty"`
}

type NodeJSON struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Hostname      string         `json:"hostname"`
		Addr          string         `json:"addr"`
		Version       string         `json:"version"`
		CloudMetadata *CloudMetadata `json:"cloud_metadata,omitempty"`
	} `json:"spec"`
}

type ParsedName struct {
	Env     string
	Service string
	IP      string
}

func Login(proxy string) error {
	cmd := exec.Command("tsh", "login", "--proxy="+proxy)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error executing command: %v, stderr: %s", err, string(output))
	}
	fmt.Println(string(output))
	return nil
}

func GetNodes(proxy string) ([]Node, error) {
	cmd := exec.Command("tctl", "nodes", "ls", "--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if loginErr := Login(proxy); loginErr != nil {
			return nil, fmt.Errorf("Login failed: %v", loginErr)
		}
		// Retry the command after successful login
		cmd = exec.Command("tctl", "nodes", "ls", "--format=json") // <-- create a new Cmd
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("Error executing command after login: %v, stderr: %s", err, string(output))
		}
	}

	// Parse the JSON output
	var nodesJSON []NodeJSON
	if err := json.Unmarshal(output, &nodesJSON); err != nil {
		return nil, fmt.Errorf("Error parsing JSON output: %v", err)
	}

	// Convert to the desired Node struct
	var nodes []Node
	for _, nodeJSON := range nodesJSON {
		node := Node{
			UUID:          nodeJSON.Metadata.Name,
			Labels:        nodeJSON.Metadata.Labels,
			Hostname:      nodeJSON.Spec.Hostname,
			Addr:          nodeJSON.Spec.Addr,
			Version:       nodeJSON.Spec.Version,
			Env:           nodeJSON.Metadata.Labels["env"],
			CloudMetadata: nodeJSON.Spec.CloudMetadata,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func ParseName(origName string) ParsedName {
	pattern := `^(?P<env>[^-]+)-nc2-(?P<service>[^-]+(?:-[^-]+)*)-(?P<ip>\d+-\d+-\d+-\d+)$`
	re := regexp.MustCompile(pattern)

	// Match the pattern and extract the components
	match := re.FindStringSubmatch(origName)
	if match == nil {
		return ParsedName{}
	}

	// Extract the named groups
	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	// Extract env, service, and IP address
	env := result["env"]
	service := result["service"]
	ip := result["ip"]

	return ParsedName{Env: env, Service: service, IP: ip}
}

func GroupNodesByHostname(nodes []Node) map[string][]Node {
	grouped := make(map[string][]Node)
	for _, node := range nodes {
		grouped[node.Hostname] = append(grouped[node.Hostname], node)
	}
	// Sort the nodes in each group by Hostname
	for _, group := range grouped {
		sort.Slice(group, func(i, j int) bool {
			return group[i].Hostname < group[j].Hostname
		})
	}
	return grouped
}

func GroupNodesByEnvAndService(nodes []Node) map[string]map[string][]Node {
	grouped := make(map[string]map[string][]Node)
	for _, node := range nodes {
		env := node.Env
		service := node.Labels["service"]
		if grouped[env] == nil {
			grouped[env] = make(map[string][]Node)
		}
		grouped[env][service] = append(grouped[env][service], node)
	}
	return grouped
}

// AssignZoneAliases groups nodes by env+service, sorts each group by the 3rd
// octet of the embedded IP address, and sets NewHostname to a human-friendly
// alias of the form "<env>-nc2-<service>-<a|b|c>" on every node that can be
// parsed. Nodes whose hostname does not match the expected pattern are left
// unchanged. The modified slice is returned.
func AssignZoneAliases(nodes []Node) []Node {
	zones := []string{"a", "b", "c"}

	// Collect indices per env+service key.
	grouped := make(map[string][]int)
	for i, node := range nodes {
		parsed := ParseName(node.Hostname)
		if parsed.Env == "" {
			continue
		}
		key := parsed.Env + "|" + parsed.Service
		grouped[key] = append(grouped[key], i)
	}

	for _, indices := range grouped {
		// Sort by the 3rd octet (index 2) of the dash-encoded IP.
		sort.Slice(indices, func(a, b int) bool {
			ipA := strings.Split(ParseName(nodes[indices[a]].Hostname).IP, "-")
			ipB := strings.Split(ParseName(nodes[indices[b]].Hostname).IP, "-")
			if len(ipA) < 3 || len(ipB) < 3 {
				return false
			}
			thirdA, _ := strconv.Atoi(ipA[2])
			thirdB, _ := strconv.Atoi(ipB[2])
			return thirdA < thirdB
		})

		for i, idx := range indices {
			parsed := ParseName(nodes[idx].Hostname)
			zone := zones[i%len(zones)]
			nodes[idx].NewHostname = fmt.Sprintf("%s-nc2-%s-%s", parsed.Env, parsed.Service, zone)
		}
	}

	return nodes
}

func SaveNodesToFile(nodes []Node, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(nodes)
}

func LoadNodesFromFile(filename string) ([]Node, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var nodes []Node
	err = json.NewDecoder(f).Decode(&nodes)
	return nodes, err
}
