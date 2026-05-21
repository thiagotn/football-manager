package services

import (
	"errors"
	"math"
	"math/rand"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

var bibColorHex = map[string]string{
	"laranja":  "#f97316",
	"azul":     "#3b82f6",
	"verde":    "#22c55e",
	"vermelho": "#ef4444",
	"amarelo":  "#eab308",
	"preto":    "#1f2937",
	"branco":   "#f1f5f9",
}

var teamNames = []string{
	"Real Madruga", "Barcelusa", "Barsemlona", "Meia Boca Juniors",
	"Baile de Munique", "Varmeiras", "Atecubanos FC", "Inter de Limão",
	"Manchester Cachaça", "Real Matismo", "Paysanduba", "Horriver Plate",
	"Patético de Madrid", "Shakhtar dos Leks", "Espressinho da Mooca",
}

var teamColors = []string{
	"#e53e3e", "#3b82f6", "#f59e0b", "#22c55e", "#f97316",
	"#a855f7", "#ec4899", "#06b6d4", "#84cc16", "#14b8a6",
}

// TeamResult holds the output of the team builder for one team.
type TeamResult struct {
	Name       string             `json:"name"`
	Color      string             `json:"color"`
	Position   int                `json:"position"`
	Players    []db.PlayerForDraw `json:"players"`
	SkillTotal int                `json:"skill_total"`
}

func pickNames(n int) []string {
	pool := make([]string, len(teamNames))
	copy(pool, teamNames)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	names := make([]string, 0, n)
	seen := make(map[string]bool)
	for _, name := range pool {
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
		if len(names) == n {
			break
		}
	}
	for idx := 2; len(names) < n; idx++ {
		candidate := pool[0] + " " + string(rune('0'+idx))
		if !seen[candidate] {
			names = append(names, candidate)
			seen[candidate] = true
		}
	}
	return names
}

func optimizeTeams(teams [][]db.PlayerForDraw) {
	for iter := 0; iter < 500; iter++ {
		totals := make([]int, len(teams))
		for i, t := range teams {
			for _, p := range t {
				totals[i] += p.SkillStars
			}
		}
		maxT, minT := totals[0], totals[0]
		for _, v := range totals {
			if v > maxT {
				maxT = v
			}
			if v < minT {
				minT = v
			}
		}
		if maxT-minT <= 1 {
			break
		}

		gkCounts := make([]int, len(teams))
		for i, t := range teams {
			for _, p := range t {
				if p.Position == "gk" {
					gkCounts[i]++
				}
			}
		}

		bestImprovement := 0
		bestA, bestI, bestB, bestJ := -1, -1, -1, -1
		currentSpread := maxT - minT

		for a := 0; a < len(teams); a++ {
			for b := a + 1; b < len(teams); b++ {
				for i, pa := range teams[a] {
					for j, pb := range teams[b] {
						if pa.SkillStars == pb.SkillStars {
							continue
						}
						paGK := pa.Position == "gk"
						pbGK := pb.Position == "gk"
						if paGK != pbGK {
							if paGK && gkCounts[a] <= 1 {
								continue
							}
							if pbGK && gkCounts[b] <= 1 {
								continue
							}
						}
						newA := totals[a] - pa.SkillStars + pb.SkillStars
						newB := totals[b] - pb.SkillStars + pa.SkillStars
						newMax, newMin := newA, newA
						for k, v := range totals {
							var vv int
							switch k {
							case a:
								vv = newA
							case b:
								vv = newB
							default:
								vv = v
							}
							if vv > newMax {
								newMax = vv
							}
							if vv < newMin {
								newMin = vv
							}
						}
						improvement := currentSpread - (newMax - newMin)
						if improvement > bestImprovement {
							bestImprovement = improvement
							bestA, bestI, bestB, bestJ = a, i, b, j
						}
					}
				}
			}
		}
		if bestA == -1 {
			break
		}
		teams[bestA][bestI], teams[bestB][bestJ] = teams[bestB][bestJ], teams[bestA][bestI]
	}
}

// BuildTeams implements the snake-draft + greedy-swap algorithm ported from Python.
// confirmed: slice of confirmed players with position and skill_stars.
// playersPerTeam: field players per team (GK not counted).
// teamSlots: optional custom name/color configuration from the group.
func BuildTeams(confirmed []db.PlayerForDraw, playersPerTeam int, teamSlots []db.TeamSlot) ([]TeamResult, []db.PlayerForDraw, error) {
	teamSize := playersPerTeam + 1
	nTeams := int(math.Ceil(float64(len(confirmed)) / float64(teamSize)))
	if nTeams < 2 {
		return nil, nil, errors.New("confirmed insuficientes para montar times")
	}

	// Shuffle then group by position
	pool := make([]db.PlayerForDraw, len(confirmed))
	copy(pool, confirmed)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	byPos := make(map[string][]db.PlayerForDraw)
	for _, p := range pool {
		pos := p.Position
		if pos == "" {
			pos = "mei"
		}
		byPos[pos] = append(byPos[pos], p)
	}
	for pos := range byPos {
		ps := byPos[pos]
		rand.Shuffle(len(ps), func(i, j int) { ps[i], ps[j] = ps[j], ps[i] })
		// sort by skill desc (stable within same skill due to shuffle above)
		for i := 0; i < len(ps)-1; i++ {
			for j := i + 1; j < len(ps); j++ {
				if ps[j].SkillStars > ps[i].SkillStars {
					ps[i], ps[j] = ps[j], ps[i]
				}
			}
		}
		byPos[pos] = ps
	}

	gks := byPos["gk"]
	delete(byPos, "gk")

	teams := make([][]db.PlayerForDraw, nTeams)
	for i := range teams {
		teams[i] = []db.PlayerForDraw{}
	}
	var overflow []db.PlayerForDraw

	assignTiers := func(group []db.PlayerForDraw, perTeam int) {
		toDist := group
		if len(toDist) > perTeam*nTeams {
			overflow = append(overflow, toDist[perTeam*nTeams:]...)
			toDist = toDist[:perTeam*nTeams]
		}
		for round := 0; round < perTeam; round++ {
			start := round * nTeams
			end := start + nTeams
			if end > len(toDist) {
				end = len(toDist)
			}
			tier := make([]db.PlayerForDraw, end-start)
			copy(tier, toDist[start:end])
			rand.Shuffle(len(tier), func(i, j int) { tier[i], tier[j] = tier[j], tier[i] })
			for ti, p := range tier {
				teams[ti] = append(teams[ti], p)
			}
		}
	}

	// Step 1: assign GKs
	gksForTeams := gks
	if len(gksForTeams) > nTeams {
		overflow = append(overflow, gksForTeams[nTeams:]...)
		gksForTeams = gksForTeams[:nTeams]
	}
	rand.Shuffle(len(gksForTeams), func(i, j int) { gksForTeams[i], gksForTeams[j] = gksForTeams[j], gksForTeams[i] })
	for ti, gk := range gksForTeams {
		teams[ti] = append(teams[ti], gk)
	}

	// Step 2: per-position allocation
	fieldSlots := teamSize - 1
	positions := []string{"lat", "zag", "mei", "ata"}
	posPerTeam := make(map[string]int, len(positions))
	for _, pos := range positions {
		if g := byPos[pos]; len(g) > 0 {
			posPerTeam[pos] = len(g) / nTeams
		}
	}
	for {
		total := 0
		for _, v := range posPerTeam {
			total += v
		}
		if total <= fieldSlots {
			break
		}
		maxPos := ""
		maxVal := 0
		for _, pos := range positions {
			if posPerTeam[pos] > maxVal {
				maxVal = posPerTeam[pos]
				maxPos = pos
			}
		}
		posPerTeam[maxPos]--
	}

	for _, pos := range positions {
		if g := byPos[pos]; len(g) > 0 {
			assignTiers(g, posPerTeam[pos])
		}
	}

	// Step 3: overflow fills remaining slots
	remaining := make([]int, nTeams)
	for i, t := range teams {
		remaining[i] = teamSize - len(t)
	}

	var finalReserves []db.PlayerForDraw

	// GK overflow: only to teams without a GK
	var ovGKs, ovField []db.PlayerForDraw
	for _, p := range overflow {
		if p.Position == "gk" {
			ovGKs = append(ovGKs, p)
		} else {
			ovField = append(ovField, p)
		}
	}
	rand.Shuffle(len(ovGKs), func(i, j int) { ovGKs[i], ovGKs[j] = ovGKs[j], ovGKs[i] })
	for _, gk := range ovGKs {
		assigned := false
		for i, t := range teams {
			if remaining[i] > 0 {
				hasGK := false
				for _, p := range t {
					if p.Position == "gk" {
						hasGK = true
						break
					}
				}
				if !hasGK {
					teams[i] = append(teams[i], gk)
					remaining[i]--
					assigned = true
					break
				}
			}
		}
		if !assigned {
			finalReserves = append(finalReserves, gk)
		}
	}

	// Field overflow
	rand.Shuffle(len(ovField), func(i, j int) { ovField[i], ovField[j] = ovField[j], ovField[i] })
	// sort by skill desc (stable: shuffle first)
	for i := 0; i < len(ovField)-1; i++ {
		for j := i + 1; j < len(ovField); j++ {
			if ovField[j].SkillStars > ovField[i].SkillStars {
				ovField[i], ovField[j] = ovField[j], ovField[i]
			}
		}
	}
	idx := 0
	for idx < len(ovField) {
		var openTeams []int
		for i, r := range remaining {
			if r > 0 {
				openTeams = append(openTeams, i)
			}
		}
		if len(openTeams) == 0 {
			break
		}
		end := idx + len(openTeams)
		if end > len(ovField) {
			end = len(ovField)
		}
		batch := make([]db.PlayerForDraw, end-idx)
		copy(batch, ovField[idx:end])
		rand.Shuffle(len(batch), func(i, j int) { batch[i], batch[j] = batch[j], batch[i] })
		for k, p := range batch {
			if k < len(openTeams) {
				ti := openTeams[k]
				teams[ti] = append(teams[ti], p)
				remaining[ti]--
			}
		}
		idx += len(batch)
	}
	finalReserves = append(finalReserves, ovField[idx:]...)

	// Optimization: greedy swap
	optimizeTeams(teams)

	randomNames := pickNames(nTeams)
	defaultColors := make([]string, nTeams)
	for i := range defaultColors {
		defaultColors[i] = teamColors[i%len(teamColors)]
	}

	result := make([]TeamResult, nTeams)
	for i, players := range teams {
		pos := i + 1
		var slot *db.TeamSlot
		if teamSlots != nil && i < len(teamSlots) {
			slot = &teamSlots[i]
		}

		var name, color string
		if slot != nil && slot.Name != nil && *slot.Name != "" {
			name = *slot.Name
		} else if slot != nil && slot.Color != nil && *slot.Color != "" {
			// Capitalize first letter
			s := *slot.Color
			if len(s) > 0 {
				name = "Time " + string(s[0]-32) + s[1:]
			} else {
				name = randomNames[i]
			}
		} else {
			name = randomNames[i]
		}

		if slot != nil && slot.Color != nil {
			if hex, ok := bibColorHex[*slot.Color]; ok {
				color = hex
			} else {
				color = defaultColors[i]
			}
		} else {
			color = defaultColors[i]
		}

		skillTotal := 0
		for _, p := range players {
			skillTotal += p.SkillStars
		}

		result[i] = TeamResult{
			Name:       name,
			Color:      color,
			Position:   pos,
			Players:    players,
			SkillTotal: skillTotal,
		}
	}

	return result, finalReserves, nil
}
