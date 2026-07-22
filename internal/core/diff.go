package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"

	"github.com/bladeacer/ocd/internal/models"
)

var diffCSSDir = ".obsidian_cache/css"

func DiffCSS(versionA, versionB string) *models.DiffResult {
	pathA := diffCSSDir + "/" + versionA + "/app.css"
	pathB := diffCSSDir + "/" + versionB + "/app.css"

	contentA, err := os.ReadFile(pathA)
	if err != nil {
		return &models.DiffResult{
			VersionA: versionA,
			VersionB: versionB,
			Error:    fmt.Errorf("read %s app.css: %w", versionA, err),
		}
	}

	contentB, err := os.ReadFile(pathB)
	if err != nil {
		return &models.DiffResult{
			VersionA: versionA,
			VersionB: versionB,
			Error:    fmt.Errorf("read %s app.css: %w", versionB, err),
		}
	}

	edits := myers.ComputeEdits(
		span.URIFromPath(pathA),
		string(contentA),
		string(contentB),
	)
	diff := fmt.Sprint(gotextdiff.ToUnified(
		fmt.Sprintf("v%s", versionA),
		fmt.Sprintf("v%s", versionB),
		string(contentA),
		edits,
	))

	hasDiff := false
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			hasDiff = true
			break
		}
	}

	return &models.DiffResult{
		VersionA: versionA,
		VersionB: versionB,
		Diff:     diff,
		HasDiff:  hasDiff,
	}
}
