package synapse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const devicesMarker = "devices:"
const reverseReadBlockSize = 64 * 1024

type localizedName map[string]string

type rawDevice struct {
	SerialNumber string        `json:"serialNumber"`
	Name         localizedName `json:"name"`
	ProductName  localizedName `json:"productName"`
	Category     string        `json:"category"`
	PowerStatus  *struct {
		ChargingStatus string `json:"chargingStatus"`
		Level          int    `json:"level"`
	} `json:"powerStatus"`
}

func ParseDevicesFromLog(r io.Reader) ([]Device, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)

	var latest []Device
	for scanner.Scan() {
		line := scanner.Text()
		devices, ok, err := ParseDevicesLine(line)
		if err != nil {
			return nil, err
		}
		if ok {
			latest = devices
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return latest, nil
}

func ParseLatestDevicesFromLog(r io.ReaderAt, size int64) ([]Device, error) {
	if size == 0 {
		return nil, nil
	}

	var pending []byte
	var latestParseErr error
	for offset := size; offset > 0; {
		readSize := int64(reverseReadBlockSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		chunk := make([]byte, readSize)
		n, err := r.ReadAt(chunk, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		combined := append(chunk[:n], pending...)
		lines := bytes.Split(combined, []byte{'\n'})

		start := 0
		if offset > 0 {
			pending = append([]byte(nil), lines[0]...)
			start = 1
		}

		for i := len(lines) - 1; i >= start; i-- {
			line := strings.TrimRight(string(lines[i]), "\r")
			devices, ok, err := ParseDevicesLine(line)
			if err != nil {
				if strings.Contains(line, devicesMarker) && latestParseErr == nil {
					latestParseErr = err
				}
				continue
			}
			if ok {
				return devices, nil
			}
		}
	}

	if len(pending) > 0 {
		line := strings.TrimRight(string(pending), "\r")
		devices, ok, err := ParseDevicesLine(line)
		if err != nil {
			if latestParseErr == nil {
				latestParseErr = err
			}
		} else if ok {
			return devices, nil
		}
	}

	if latestParseErr != nil {
		return nil, latestParseErr
	}
	return nil, nil
}

func ParseDevicesLine(line string) ([]Device, bool, error) {
	idx := strings.Index(line, devicesMarker)
	if idx == -1 {
		return nil, false, nil
	}

	payload := strings.TrimSpace(line[idx+len(devicesMarker):])
	start := strings.IndexByte(payload, '[')
	if start == -1 {
		return nil, false, nil
	}

	var raw []rawDevice
	decoder := json.NewDecoder(bytes.NewBufferString(payload[start:]))
	if err := decoder.Decode(&raw); err != nil {
		return nil, false, fmt.Errorf("parse devices json: %w", err)
	}

	devices := make([]Device, 0, len(raw))
	for _, item := range raw {
		if item.PowerStatus == nil {
			continue
		}
		devices = append(devices, Device{
			SerialNumber:   item.SerialNumber,
			Name:           bestName(item.Name, item.ProductName),
			Category:       item.Category,
			BatteryLevel:   item.PowerStatus.Level,
			ChargingStatus: item.PowerStatus.ChargingStatus,
		})
	}

	return devices, true, nil
}

func bestName(names ...localizedName) string {
	for _, name := range names {
		if value := strings.TrimSpace(name["en"]); value != "" {
			return value
		}
		for _, value := range name {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return "Unknown Razer device"
}
