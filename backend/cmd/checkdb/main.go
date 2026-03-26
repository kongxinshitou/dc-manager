package main

import (
	"fmt"
	"dcmanager/database"
	"dcmanager/models"
	"os"
)

func main() {
	dbPath := "dc_manager.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	database.Init(dbPath)

	// Check columns via raw query
	var result []struct {
		ID       uint
		UPos     string
		StartU   *int
		EndU     *int
	}
	database.DB.Raw("SELECT id, u_position, start_u, end_u FROM devices LIMIT 20").Scan(&result)
	fmt.Println("=== Sample device U values ===")
	for _, r := range result {
		su := "NULL"
		eu := "NULL"
		if r.StartU != nil {
			su = fmt.Sprintf("%d", *r.StartU)
		}
		if r.EndU != nil {
			eu = fmt.Sprintf("%d", *r.EndU)
		}
		fmt.Printf("  id=%d u_position=%q start_u=%s end_u=%s\n", r.ID, r.UPos, su, eu)
	}

	var total, nonNull int64
	database.DB.Model(&models.Device{}).Count(&total)
	database.DB.Model(&models.Device{}).Where("start_u IS NOT NULL").Count(&nonNull)
	fmt.Printf("\nTotal devices: %d, With start_u: %d\n", total, nonNull)

	// Check cabinet endpoint data
	var datacenters []string
	database.DB.Model(&models.Device{}).Distinct("datacenter").Where("datacenter != ''").Pluck("datacenter", &datacenters)
	fmt.Printf("\nDatacenters: %v\n", datacenters)

	if len(datacenters) > 0 {
		var cabinets []string
		database.DB.Model(&models.Device{}).Where("datacenter = ? AND cabinet != ''", datacenters[0]).Distinct("cabinet").Order("cabinet").Pluck("cabinet", &cabinets)
		fmt.Printf("Cabinets for %q: %v\n", datacenters[0], cabinets)
	}
}
