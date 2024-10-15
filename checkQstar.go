package main

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
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
		return nil, err
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
				return nil, errors.New("failed to read `page_size`")
			}
			item["page_size"] = i
			pageSize = i
		}

		// replicas
		if strings.HasPrefix(line, "Replicas:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `replicas`")
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

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// cache_root
		if strings.HasPrefix(line, "Cache root:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `cache_root`")
			}
			item["cache_root"] = s
			continue
		}

		// mount_point
		if strings.HasPrefix(line, "Mount point:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `mount_point`")
			}
			item["mount_point"] = s
			continue
		}

		// max_number_of_pages
		if strings.HasPrefix(line, "Max number of pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `max_number_of_pages`")
			}
			item["max_number_of_pages"] = i
			continue
		}

		// low_primary_capacity
		if strings.HasPrefix(line, "Low primary capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `low_primary_capacity`")
			}
			item["low_primary_capacity_pages"] = i
			item["low_primary_capacity_bytes"] = i * pageSize
			continue
		}

		// high_primary_capacity
		if strings.HasPrefix(line, "High primary capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `high_primary_capacity`")
			}
			item["high_primary_capacity_pages"] = i
			item["high_primary_capacity_bytes"] = i * pageSize
			continue
		}

		// read_reserved_capacity
		if strings.HasPrefix(line, "Read reserved capacity:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `read_reserved_capacity`")
			}
			item["read_reserved_capacity_pages"] = i
			item["read_reserved_capacity_bytes"] = i * pageSize
			continue
		}

		// prefetch_priority_period
		if strings.HasPrefix(line, "Prefetch priority period:") {
			b, err := getBool(line)
			if err != nil {
				return nil, errors.New("failed to read `prefetch_priority_period`")
			}
			item["prefetch_priority_period"] = b
			continue
		}

		// prefetching_mode
		if strings.HasPrefix(line, "Prefetching mode:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `prefetching_mode`")
			}
			item["prefetching_mode"] = s
			continue
		}

		// cold_prefetching
		if strings.HasPrefix(line, "Cold prefetching:") {
			b, err := getBool(line)
			if err != nil {
				return nil, errors.New("failed to read `cold_prefetching`")
			}
			item["cold_prefetching"] = b
			continue
		}

		// cache_write_throttling
		if strings.HasPrefix(line, "Cache write throttling:") {
			b, err := getBool(line)
			if err != nil {
				return nil, errors.New("failed to read `cache_write_throttling`")
			}
			item["cache_write_throttling"] = b
			continue
		}

		// automatic_keep_in_cache
		if strings.HasPrefix(line, "Automatic keep in cache:") {
			b, err := getBool(line)
			if err != nil {
				return nil, errors.New("failed to read `automatic_keep_in_cache`")
			}
			item["automatic_keep_in_cache"] = b
			continue
		}

		// present_pages
		if strings.HasPrefix(line, "Present pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `present_pages`")
			}
			item["present_pages"] = i
			item["present_bytes"] = i * pageSize
			continue
		}

		// primary_pages
		if strings.HasPrefix(line, "Primary pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `primary_pages`")
			}
			item["primary_pages"] = i
			item["primary_bytes"] = i * pageSize
			continue
		}

		// replicated_pages
		if strings.HasPrefix(line, "Replicated pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `replicated_pages`")
			}
			item["replicated_pages"] = i
			item["replicated_bytes"] = i * pageSize
			continue
		}

		// archived_pages
		if strings.HasPrefix(line, "Archived pages:") {
			i, err := getPages(line)
			if err != nil {
				return nil, errors.New("failed to read `archived_pages`")
			}
			item["archived_pages"] = i
			item["archived_bytes"] = i * pageSize
			continue
		}

		// keep_in_cache
		if strings.HasPrefix(line, "Keep in cache:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `keep_in_cache`")
			}
			item["keep_in_cache"] = i
		}

		// archived_since_mount
		if strings.HasPrefix(line, "Archived since mount:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, errors.New("failed to read `archived_since_mount`")
			}
			item["archived_since_mount"] = i
			continue
		}

		// replicated_since_mount
		if strings.HasPrefix(line, "Replicated since mount:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, errors.New("failed to read `replicated_since_mount`")
			}
			item["replicated_since_mount"] = i
		}

		// files_in_cache
		if strings.HasPrefix(line, "Files in cache:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `files_in_cache`")
			}
			item["files_in_cache"] = i
			continue
		}

		// directories
		if strings.HasPrefix(line, "Directories:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `directories`")
			}
			item["directories"] = i
			continue
		}

		// streams
		if strings.HasPrefix(line, "Streams:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `streams`")
			}
			item["streams"] = i
			continue
		}

		// number_of_delayed_events
		if strings.HasPrefix(line, "Number of delayed events:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `number_of_delayed_events`")
			}
			item["number_of_delayed_events"] = i
			continue
		}

		// read_write_access
		if strings.HasPrefix(line, "Read/write access:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `read_write_access`")
			}
			item["read_write_access"] = s
			continue
		}

		// archiving
		if strings.HasPrefix(line, "Archiving:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `archiving`")
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
				return nil, errors.New("failed to read `medium_drive_type`")
			}
			(*replica)["medium_drive_type"] = s
			continue
		}

		// extent_size
		if strings.HasPrefix(line, "Extent size:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, errors.New("failed to read `extent_size`")
			}
			(*replica)["extent_size"] = i
			continue
		}

		// write_pool_count
		if strings.HasPrefix(line, "Write pool count:") {
			i, err := getInt64(line)
			if err != nil {
				return nil, errors.New("failed to read `write_pool_count`")
			}
			(*replica)["write_pool_count"] = i
			continue
		}

		// last_write_on
		if strings.HasPrefix(line, "Last write on:") {
			s, err := getString(line)
			if err != nil {
				return nil, errors.New("failed to read `last_write_on`")
			}
			(*replica)["last_write_on"] = s
			continue
		}

		// free_space_on_current_partition
		if strings.HasPrefix(line, "Free space on current partition:") {
			_, i, err := getSize(line)
			if err != nil {
				return nil, errors.New("failed to read `free_space_on_current_partition`")
			}
			(*replica)["free_space_on_current_partition"] = i
			continue
		}

		// compression
		if strings.HasPrefix(line, "Compression:") {
			b, err := getBool(line)
			if err != nil {
				return nil, errors.New("failed to read `compression`")
			}
			(*replica)["compression"] = b
			continue
		}
	}

	return &params{
		params:   item,
		replicas: replicas,
	}, nil
}

func CheckQstar() (map[string][]map[string]any, error) {
	state := map[string][]map[string]any{}

	// out, err := exec.Command("bash", "-c", "cat df.output.example.txt").Output()
	out, err := exec.Command("bash", "-c", "df -t fuse.mcfs").Output()
	if err != nil {
		return nil, err
	}

	filesystems := []map[string]any{}
	replicas := []map[string]any{}
	lines := reSplit.Split(string(out), -1)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Filesystem") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			res, err := readFilesystem(fields[0])
			if err != nil {
				return nil, err
			}
			filesystems = append(filesystems, res.params)
			replicas = append(replicas, res.replicas...)
		}
	}

	state["filesystems"] = filesystems
	state["replicas"] = replicas

	// Print debug dump
	// b, _ := json.MarshalIndent(state, "", "    ")
	// log.Fatal(string(b))

	return state, nil
}
