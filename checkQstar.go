package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/infrasonar/go-libagent"
)

var reSplit = regexp.MustCompile(`\r?\n`)
var reNumber = regexp.MustCompile(`\d+$`)
var rePages = regexp.MustCompile(`(\d+)(\s*pages)?$`)
var reSize = regexp.MustCompile(`(.*)(\s|\:)([\d\.]+)\s*([TGMK])iB$`)
var reString = regexp.MustCompile(`\:\s*(\S.*)$`)
var reReplica = regexp.MustCompile(`^Replica\s(\d+)\s*\:(.*)$`)

func getInt64(line string) (int64, error) {
	matches := reNumber.FindStringSubmatch(line)
	if len(matches) != 1 {
		return 0, errors.New("no match for int")
	}
	return strconv.ParseInt(matches[0], 10, 64)
}

func getSize(line string) (string, int64, error) {
	matches := reSize.FindStringSubmatch(line)
	if len(matches) != 5 {
		return line, 0, errors.New("no match for size")
	}
	line = matches[1]
	f, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return line, 0, err
	}

	switch matches[4] {
	case "T":
		f *= 1024
		fallthrough
	case "G":
		f *= 1024
		fallthrough
	case "M":
		f *= 1024
		fallthrough
	case "K":
		f *= 1024
	}

	return strings.TrimSpace(line), int64(f), err
}

func getPages(line string) (int64, error) {
	line, _, _ = getSize(line) // strip bytes
	matches := rePages.FindStringSubmatch(line)
	if len(matches) != 3 {
		return 0, errors.New("no match for pages")
	}
	return strconv.ParseInt(matches[1], 10, 64)
}

func getBool(line string) (bool, error) {
	if strings.HasSuffix(line, "Yes") || strings.HasSuffix(line, "On") {
		return true, nil
	}
	if strings.HasSuffix(line, "No") || strings.HasSuffix(line, "Off") {
		return false, nil
	}
	return false, errors.New("failed to read boolean Yes/No")
}

func getString(line string) (string, error) {
	matches := reString.FindStringSubmatch(line)
	if len(matches) != 2 {
		return "", errors.New("no match for string")
	}
	return matches[1], nil
}

type params struct {
	params   map[string]any
	replicas []map[string]any
}

func readFilesystem(filesystem string) (*params, error) {
	// out, err := exec.Command("bash", "-c", "cat output.example.txt").Output()
	out, err := exec.Command("bash", "-c", fmt.Sprintf("mmparam %s", filesystem)).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute `mmparam` (%s)", err)
	}

	lines := reSplit.Split(string(out), -1)

	item := map[string]any{
		"name": filesystem,
	}

	var pageSize int64 = 0
	var numReplicas int64 = 0

	for _, line := range lines {
		// page_size
		if strings.HasPrefix(line, "Page size:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `page_size` (%s)", err)
			}
			item["page_size"] = i
			pageSize = i
		}

		// replicas
		if strings.HasPrefix(line, "Replicas:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `replicas` (%s)", err)
			}
			item["replicas"] = i
			numReplicas = i
		}
	}

	if pageSize == 0 {
		return nil, errors.New("missing required page size")
	}

	replicas := make([]map[string]any, numReplicas)
	for i := range numReplicas {
		replicas[i] = map[string]any{
			"name":       fmt.Sprintf("%s-replica-%d", filesystem, i),
			"replica":    fmt.Sprintf("Replica %d", i),
			"filesystem": filesystem,
		}
	}

	var replica *map[string]any = nil
	var present_pages int64 = -1
	var max_number_of_pages int64 = -1

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// cache_root
		if strings.HasPrefix(line, "Cache root:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `cache_root` (%s)", err)
			}
			item["cache_root"] = s
			continue
		}

		// mount_point
		if strings.HasPrefix(line, "Mount point:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `mount_point` (%s)", err)
			}
			item["mount_point"] = s
			continue
		}

		// max_number_of_pages
		if strings.HasPrefix(line, "Max number of pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `max_number_of_pages` (%s)", err)
			}
			max_number_of_pages = i
			item["max_number_of_pages"] = i
			continue
		}

		// low_primary_capacity
		if strings.HasPrefix(line, "Low primary capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `low_primary_capacity` (%s)", err)
			}
			item["low_primary_capacity_pages"] = i
			item["low_primary_capacity_bytes"] = i * pageSize
			continue
		}

		// high_primary_capacity
		if strings.HasPrefix(line, "High primary capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `high_primary_capacity` (%s)", err)
			}
			item["high_primary_capacity_pages"] = i
			item["high_primary_capacity_bytes"] = i * pageSize
			continue
		}

		// read_reserved_capacity
		if strings.HasPrefix(line, "Read reserved capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `read_reserved_capacity` (%s)", err)
			}
			item["read_reserved_capacity_pages"] = i
			item["read_reserved_capacity_bytes"] = i * pageSize
			continue
		}

		// prefetch_priority_period
		if strings.HasPrefix(line, "Prefetch priority period:") {
			b, err := getBool(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `prefetch_priority_period` (%s)", err)
			}
			item["prefetch_priority_period"] = b
			continue
		}

		// prefetching_mode
		if strings.HasPrefix(line, "Prefetching mode:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `prefetching_mode` (%s)", err)
			}
			item["prefetching_mode"] = s
			continue
		}

		// cold_prefetching
		if strings.HasPrefix(line, "Cold prefetching:") {
			b, err := getBool(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `cold_prefetching` (%s)", err)
			}
			item["cold_prefetching"] = b
			continue
		}

		// cache_write_throttling
		if strings.HasPrefix(line, "Cache write throttling:") {
			b, err := getBool(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `cache_write_throttling` (%s)", err)
			}
			item["cache_write_throttling"] = b
			continue
		}

		// automatic_keep_in_cache
		if strings.HasPrefix(line, "Automatic keep in cache:") {
			b, err := getBool(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `automatic_keep_in_cache` (%s)", err)
			}
			item["automatic_keep_in_cache"] = b
			continue
		}

		// present_pages
		if strings.HasPrefix(line, "Present pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `present_pages` (%s)", err)
			}
			present_pages = i
			item["present_pages"] = i
			item["present_bytes"] = i * pageSize
			continue
		}

		// primary_pages
		if strings.HasPrefix(line, "Primary pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `primary_pages` (%s)", err)
			}
			item["primary_pages"] = i
			item["primary_bytes"] = i * pageSize
			continue
		}

		// replicated_pages (optional)
		if strings.HasPrefix(line, "Replicated pages:") {
			i, err := getPages(line)
			if err == nil {
				item["replicated_pages"] = i
				item["replicated_bytes"] = i * pageSize
			}
			continue
		}

		// archived_pages (optional)
		if strings.HasPrefix(line, "Archived pages:") {
			i, err := getPages(line)
			if err == nil {
				item["archived_pages"] = i
				item["archived_bytes"] = i * pageSize
			}
			continue
		}

		// keep_in_cache
		if strings.HasPrefix(line, "Keep in cache:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `keep_in_cache` (%s)", err)
			}
			item["keep_in_cache"] = i
		}

		// archived_since_mount
		if strings.HasPrefix(line, "Archived since mount:") {
			_, i, err := getSize(line)
			if err == nil {
				item["archived_since_mount"] = i
			}
			continue
		}

		// replicated_since_mount
		if strings.HasPrefix(line, "Replicated since mount:") {
			_, i, err := getSize(line)
			if err == nil {
				item["replicated_since_mount"] = i
			}
			continue
		}

		// files_in_cache
		if strings.HasPrefix(line, "Files in cache:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `files_in_cache` (%s)", err)
			}
			item["files_in_cache"] = i
			continue
		}

		// directories
		if strings.HasPrefix(line, "Directories:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `directories` (%s)", err)
			}
			item["directories"] = i
			continue
		}

		// streams
		if strings.HasPrefix(line, "Streams:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `streams` (%s)", err)
			}
			item["streams"] = i
			continue
		}

		// number_of_delayed_events
		if strings.HasPrefix(line, "Number of delayed events:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `number_of_delayed_events` (%s)", err)
			}
			item["number_of_delayed_events"] = i
			continue
		}

		// read_write_access
		if strings.HasPrefix(line, "Read/write access:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `read_write_access` (%s)", err)
			}
			item["read_write_access"] = s
			continue
		}

		// archiving
		if strings.HasPrefix(line, "Archiving:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `archiving` (%s)", err)
			}
			item["archiving"] = s
			continue
		}

		//
		// Replica metrics
		//
		rMatches := reReplica.FindStringSubmatch(line)
		if len(rMatches) == 3 {
			i, err := strconv.Atoi(rMatches[1])
			if err != nil {
				return nil, errors.New("failed to read replica number")
			}
			if i < 0 || i >= len(replicas) {
				return nil, errors.New("replica out of range")
			}
			replica = &replicas[i]

			rest := strings.Split(strings.TrimSpace(rMatches[2]), ", ")
			if len(rest) >= 1 {
				key := rest[0]
				(*replica)["key"] = key
				fields := strings.Split(key, "-")
				if len(fields) == 3 {
					(*replica)["location"] = fields[0]
					(*replica)["share"] = fields[1]
					(*replica)["local_intergral_volume"] = fields[2]
				} else if len(fields) == 2 {
					(*replica)["location"] = fields[0]
					(*replica)["local_intergral_volume"] = fields[1]
				}
			}
			online := false
			inSync := false
			isRead := false
			for _, f := range rest {
				if f == "online" {
					online = true
				} else if f == "read" {
					isRead = true
				} else if f == "in sync" {
					inSync = true
				}
			}
			(*replica)["online"] = online
			(*replica)["in_sync"] = inSync
			(*replica)["read"] = isRead
		}

		if replica == nil {
			continue // al metrics below are for replicas
		}

		// migrator
		if strings.HasPrefix(line, "Migrator:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `migrator`")
			}
			(*replica)["migrator"] = s
			continue
		}

		// medium_drive_type
		if strings.HasPrefix(line, "Medium drive type:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `medium_drive_type` (%s)", err)
			}
			(*replica)["medium_drive_type"] = s
			continue
		}

		// extent_size
		if strings.HasPrefix(line, "Extent size:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `extent_size` (%s)", err)
			}
			(*replica)["extent_size"] = i
			continue
		}

		// write_pool_count
		if strings.HasPrefix(line, "Write pool count:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `write_pool_count` (%s)", err)
			}
			(*replica)["write_pool_count"] = i
			continue
		}

		// last_write_on
		if strings.HasPrefix(line, "Last write on:") {
			s, err := getString(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `last_write_on` (%s)", err)
			}
			(*replica)["last_write_on"] = s
			continue
		}

		// free_space_on_current_partition
		if strings.HasPrefix(line, "Free space on current partition:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `free_space_on_current_partition` (%s)", err)
			}
			(*replica)["free_space_on_current_partition"] = i
			continue
		}

		// compression
		if strings.HasPrefix(line, "Compression:") {
			b, err := getBool(line)
			if err != nil {
				return nil, fmt.Errorf("failed to read `compression` (%s)", err)
			}
			(*replica)["compression"] = b
			continue
		}
	}

	// calculate free bytes
	if max_number_of_pages >= 0 && present_pages >= 0 {
		free_pages := max_number_of_pages - present_pages
		if free_pages >= 0 {
			item["free_pages"] = free_pages
			item["free_bytes"] = free_pages * pageSize
		}
	}

	return &params{
		params:   item,
		replicas: replicas,
	}, nil
}

func CheckQstar(_ *libagent.Check) (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	// out, err := exec.Command("bash", "-c", "cat df.output.example.txt").Output()
	out, err := exec.Command("bash", "-c", "df -t fuse.mcfs").Output()

	filesystems := []map[string]any{}
	replicas := []map[string]any{}

	if err == nil {
		lines := reSplit.Split(string(out), -1)

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "Filesystem") || line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) > 0 {
				// Get the first in line
				first := fields[0]
				// Read filesystem
				res, err := readFilesystem(first)
				if err != nil {
					return nil, err
				}

				filesystems = append(filesystems, res.params)
				replicas = append(replicas, res.replicas...)
			}
		}
	} else {
		log.Printf("Failed to execute: bash -c \"df -t fuse.mcfs\" (%v)\n", err)
	}

	state["filesystems"] = filesystems
	state["replicas"] = replicas

	// Add the agent version
	state["agent"] = []map[string]any{{
		"name":    "qstar",
		"version": version,
	}}

	// Print debug dump
	// b, _ := json.MarshalIndent(state, "", "    ")
	// log.Fatal(string(b))

	return state, nil
}
