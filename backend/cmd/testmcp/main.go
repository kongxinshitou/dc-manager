package main

import (
	"dcmanager/database"
	"dcmanager/models"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	uRangeRe  = regexp.MustCompile(`^(\d+)\s*[-~]\s*(\d+)\s*[Uu]$`)
	uSingleRe = regexp.MustCompile(`^(\d+)\s*[Uu]$`)
	datacenterRe = regexp.MustCompile(`(?i)^(IDC)?\s*(\d+)\s*[-]\s*(\d+)$`)
	cabinetRe    = regexp.MustCompile(`^([A-Za-z]+)[\s\-]*(\d+)$`)
)

func normalizeDatacenter(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := datacenterRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	return fmt.Sprintf("IDC%s-%s", m[2], m[3])
}

func normalizeCabinet(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	m := cabinetRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	letter := strings.ToUpper(m[1])
	num := m[2]
	if len(num) == 1 {
		num = "0" + num
	}
	return fmt.Sprintf("%s-%s", letter, num)
}

func parseUPosition(pos string) (startU, endU *int) {
	pos = strings.TrimSpace(pos)
	if pos == "" {
		return nil, nil
	}
	if m := uRangeRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		return &a, &b
	}
	if m := uSingleRe.FindStringSubmatch(pos); m != nil {
		a, _ := strconv.Atoi(m[1])
		return &a, &a
	}
	return nil, nil
}

func main() {
	// Test normalization
	tests := []struct{ input, expected string }{
		{"A04", "A-04"},
		{"A01", "A-01"},
		{"B17", "B-17"},
		{"A-01", "A-01"},
		{"B-09", "B-09"},
		{"C1", "C-01"},
	}
	fmt.Println("=== Cabinet normalization tests ===")
	for _, t := range tests {
		result := normalizeCabinet(t.input)
		status := "OK"
		if result != t.expected {
			status = "FAIL"
		}
		fmt.Printf("  %s: %q -> %q (expected %q)\n", status, t.input, result, t.expected)
	}

	dcTests := []struct{ input, expected string }{
		{"1-2", "IDC1-2"},
		{"2-1", "IDC2-1"},
		{"IDC1-2", "IDC1-2"},
		{"idc1-1", "IDC1-1"},
	}
	fmt.Println("\n=== Datacenter normalization tests ===")
	for _, t := range dcTests {
		result := normalizeDatacenter(t.input)
		status := "OK"
		if result != t.expected {
			status = "FAIL"
		}
		fmt.Printf("  %s: %q -> %q (expected %q)\n", status, t.input, result, t.expected)
	}

	// Test DB matching
	dbPath := "dc_manager.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	database.Init(dbPath)

	fmt.Println("\n=== Device auto-match test ===")
	// Test case: IDC1-2, A-04, 15-16U
	dc := normalizeDatacenter("1-2")
	cab := normalizeCabinet("A04")
	upos := "15-16U"
	fmt.Printf("  Looking for: dc=%q, cab=%q, upos=%q\n", dc, cab, upos)

	startU, endU := parseUPosition(upos)
	fmt.Printf("  Parsed U: start=%v, end=%v\n", *startU, *endU)

	var devices []models.Device
	database.DB.Where("datacenter = ? AND cabinet = ?", dc, cab).Find(&devices)
	fmt.Printf("  Devices matching dc+cab: %d\n", len(devices))
	for _, d := range devices {
		su, eu := "", ""
		if d.StartU != nil {
			su = fmt.Sprintf("%d", *d.StartU)
		}
		if d.EndU != nil {
			eu = fmt.Sprintf("%d", *d.EndU)
		}
		fmt.Printf("    ID:%d, UPos:%q, StartU:%s, EndU:%s, Brand:%s, Model:%s\n",
			d.ID, d.UPosition, su, eu, d.Brand, d.Model)
	}

	// Also try with U overlap
	if startU != nil && endU != nil {
		var matched []models.Device
		database.DB.Where("datacenter = ? AND cabinet = ?", dc, cab).
			Where("start_u IS NOT NULL AND end_u IS NOT NULL AND start_u <= ? AND end_u >= ?", *endU, *startU).
			Find(&matched)
		fmt.Printf("  Devices matching with U overlap: %d\n", len(matched))
		for _, d := range matched {
			fmt.Printf("    ID:%d, UPos:%q, Brand:%s, Model:%s\n", d.ID, d.UPosition, d.Brand, d.Model)
		}
	}

	// Test case 2: IDC1-2, A-01, 18-19U
	dc2 := normalizeDatacenter("1-2")
	cab2 := normalizeCabinet("A01")
	upos2 := "18-19U"
	fmt.Printf("\n  Looking for: dc=%q, cab=%q, upos=%q\n", dc2, cab2, upos2)

	startU2, endU2 := parseUPosition(upos2)
	var devices2 []models.Device
	database.DB.Where("datacenter = ? AND cabinet = ?", dc2, cab2).Find(&devices2)
	fmt.Printf("  Devices matching dc+cab: %d\n", len(devices2))
	for _, d := range devices2 {
		su, eu := "", ""
		if d.StartU != nil {
			su = fmt.Sprintf("%d", *d.StartU)
		}
		if d.EndU != nil {
			eu = fmt.Sprintf("%d", *d.EndU)
		}
		fmt.Printf("    ID:%d, UPos:%q, StartU:%s, EndU:%s, Brand:%s, Model:%s\n",
			d.ID, d.UPosition, su, eu, d.Brand, d.Model)
	}

	if startU2 != nil && endU2 != nil {
		var matched2 []models.Device
		database.DB.Where("datacenter = ? AND cabinet = ?", dc2, cab2).
			Where("start_u IS NOT NULL AND end_u IS NOT NULL AND start_u <= ? AND end_u >= ?", *endU2, *startU2).
			Find(&matched2)
		fmt.Printf("  Devices matching with U overlap: %d\n", len(matched2))
		for _, d := range matched2 {
			fmt.Printf("    ID:%d, UPos:%q, Brand:%s, Model:%s\n", d.ID, d.UPosition, d.Brand, d.Model)
		}
	}
}
