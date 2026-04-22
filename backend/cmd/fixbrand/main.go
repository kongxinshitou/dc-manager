package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type deviceRow struct {
	ID    uint
	Model string
	Brand string
}

type rawBrandCount struct {
	Brand string
	Count int
}

type brandGroup struct {
	Total     int
	RawCounts map[string]int
}

type modelBrandChoice struct {
	NormalizedBrand string
	CanonicalBrand  string
}

type pendingUpdate struct {
	ID       uint
	Model    string
	OldBrand string
	NewBrand string
	Reason   string
}

var missingBrandValues = map[string]struct{}{
	"":        {},
	"-":       {},
	"--":      {},
	"n/a":     {},
	"na":      {},
	"null":    {},
	"unknown": {},
	"未知":      {},
	"无":       {},
	"未填写":     {},
	"待补充":     {},
}

func main() {
	var (
		dbPath      string
		apply       bool
		sampleLimit int
	)

	flag.StringVar(&dbPath, "db", "dc_manager.db", "SQLite database path")
	flag.BoolVar(&apply, "apply", false, "apply updates to database")
	flag.IntVar(&sampleLimit, "sample", 50, "number of sample updates to print")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	var devices []deviceRow
	if err := db.Raw("SELECT id, model, brand FROM devices WHERE TRIM(IFNULL(model, '')) != ''").Scan(&devices).Error; err != nil {
		log.Fatalf("failed to load devices: %v", err)
	}

	modelChoices, conflictModels := buildModelBrandChoices(devices)
	updates := collectUpdates(devices, modelChoices)

	fmt.Printf("Loaded %d devices with non-empty model\n", len(devices))
	fmt.Printf("Models with unique historical brand: %d\n", len(modelChoices))
	fmt.Printf("Models with conflicting historical brands: %d\n", conflictModels)
	fmt.Printf("Pending brand updates: %d\n", len(updates))

	if len(updates) == 0 {
		fmt.Println("No database changes needed.")
		return
	}

	sort.Slice(updates, func(i, j int) bool {
		if updates[i].Model == updates[j].Model {
			return updates[i].ID < updates[j].ID
		}
		return updates[i].Model < updates[j].Model
	})

	fmt.Println("\nSample updates:")
	for i, upd := range updates {
		if i >= sampleLimit {
			break
		}
		fmt.Printf("  [Device %d] model=%q brand=%q -> %q (%s)\n", upd.ID, upd.Model, upd.OldBrand, upd.NewBrand, upd.Reason)
	}

	if !apply {
		fmt.Println("\nDry run only. Re-run with -apply to write changes.")
		return
	}

	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("failed to start transaction: %v", tx.Error)
	}

	updated := 0
	for _, upd := range updates {
		if err := tx.Exec("UPDATE devices SET brand = ? WHERE id = ?", upd.NewBrand, upd.ID).Error; err != nil {
			tx.Rollback()
			log.Fatalf("failed to update device %d: %v", upd.ID, err)
		}
		updated++
	}

	if err := tx.Commit().Error; err != nil {
		log.Fatalf("failed to commit updates: %v", err)
	}

	fmt.Printf("\nApplied %d brand updates.\n", updated)
}

func buildModelBrandChoices(devices []deviceRow) (map[string]modelBrandChoice, int) {
	modelStats := make(map[string]map[string]*brandGroup)

	for _, device := range devices {
		modelKey := normalizeModel(device.Model)
		brandKey := normalizeBrand(device.Brand)
		rawBrand := strings.TrimSpace(device.Brand)
		if modelKey == "" || isMissingBrand(device.Brand) || brandKey == "" {
			continue
		}
		if _, ok := modelStats[modelKey]; !ok {
			modelStats[modelKey] = make(map[string]*brandGroup)
		}
		group, ok := modelStats[modelKey][brandKey]
		if !ok {
			group = &brandGroup{RawCounts: make(map[string]int)}
			modelStats[modelKey][brandKey] = group
		}
		group.Total++
		group.RawCounts[rawBrand]++
	}

	choices := make(map[string]modelBrandChoice)
	conflictModels := 0

	for modelKey, brandGroups := range modelStats {
		if len(brandGroups) != 1 {
			conflictModels++
			continue
		}
		for normalizedBrand, group := range brandGroups {
			choices[modelKey] = modelBrandChoice{
				NormalizedBrand: normalizedBrand,
				CanonicalBrand:  pickCanonicalRawBrand(group.RawCounts),
			}
		}
	}

	return choices, conflictModels
}

func collectUpdates(devices []deviceRow, modelChoices map[string]modelBrandChoice) []pendingUpdate {
	updates := make([]pendingUpdate, 0)

	for _, device := range devices {
		modelKey := normalizeModel(device.Model)
		choice, ok := modelChoices[modelKey]
		if !ok {
			continue
		}

		currentBrand := strings.TrimSpace(device.Brand)
		currentBrandKey := normalizeBrand(currentBrand)

		switch {
		case isMissingBrand(device.Brand):
			updates = append(updates, pendingUpdate{
				ID:       device.ID,
				Model:    strings.TrimSpace(device.Model),
				OldBrand: currentBrand,
				NewBrand: choice.CanonicalBrand,
				Reason:   "filled missing brand from historical model mapping",
			})
		case currentBrandKey == choice.NormalizedBrand && currentBrand != choice.CanonicalBrand:
			updates = append(updates, pendingUpdate{
				ID:       device.ID,
				Model:    strings.TrimSpace(device.Model),
				OldBrand: currentBrand,
				NewBrand: choice.CanonicalBrand,
				Reason:   "normalized brand casing/format to historical canonical value",
			})
		}
	}

	return updates
}

func pickCanonicalRawBrand(rawCounts map[string]int) string {
	brands := make([]rawBrandCount, 0, len(rawCounts))
	for brand, count := range rawCounts {
		brands = append(brands, rawBrandCount{Brand: brand, Count: count})
	}
	sort.Slice(brands, func(i, j int) bool {
		if brands[i].Count == brands[j].Count {
			return brands[i].Brand < brands[j].Brand
		}
		return brands[i].Count > brands[j].Count
	})
	return brands[0].Brand
}

func normalizeModel(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeBrand(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func isMissingBrand(value string) bool {
	_, ok := missingBrandValues[normalizeBrand(value)]
	return ok
}
