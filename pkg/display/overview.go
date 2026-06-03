package display

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/chenhuijun/op-cli/pkg/api"
)

const noSprintLabel = "(no sprint)"

// SprintRollup summarizes my open items in one sprint.
type SprintRollup struct {
	Name        string
	Open        int
	Blocked     int
	lastUpdated string
}

// ProjectRollup summarizes my open items in one project plus its top sprints.
type ProjectRollup struct {
	Name        string
	Open        int
	Blocked     int
	Sprints     []SprintRollup
	MoreSprints int // sprints beyond the shown top-N
	lastUpdated string
}

// OverviewModel is the cross-project rollup of my open work.
type OverviewModel struct {
	Projects      []ProjectRollup
	TotalProjects int
	MoreProjects  int // projects beyond the shown top-N
}

// BuildOverview groups open work packages by project then sprint, keeping the
// top projectsN projects (by most-recent activity) and, within each, the top
// sprintsN sprints. "Recency" is the latest updatedAt of my items in that
// project/sprint — the same signal that put the project in the result set, so
// no extra queries are needed. Empty/missing dates sort last; ties break by
// name for deterministic output.
func BuildOverview(wps []api.WorkPackage, projectsN, sprintsN int) OverviewModel {
	type sprintAgg struct {
		open, blocked int
		last          string
	}
	type projAgg struct {
		open, blocked int
		last          string
		sprints       map[string]*sprintAgg
		order         []string
	}
	projects := map[string]*projAgg{}

	for i := range wps {
		wp := &wps[i]
		pname := firstNonEmpty(wp.Links.Project.Title, wp.Links.Project.Href, "(unknown project)")
		sname := firstNonEmpty(wp.Links.Version.Title, noSprintLabel)

		p := projects[pname]
		if p == nil {
			p = &projAgg{sprints: map[string]*sprintAgg{}}
			projects[pname] = p
		}
		s := p.sprints[sname]
		if s == nil {
			s = &sprintAgg{}
			p.sprints[sname] = s
			p.order = append(p.order, sname)
		}

		blocked := strings.EqualFold(wp.Links.Status.Title, "blocked")
		p.open++
		s.open++
		if blocked {
			p.blocked++
			s.blocked++
		}
		if wp.UpdatedAt > p.last {
			p.last = wp.UpdatedAt
		}
		if wp.UpdatedAt > s.last {
			s.last = wp.UpdatedAt
		}
	}

	rollups := make([]ProjectRollup, 0, len(projects))
	for pname, p := range projects {
		sort.SliceStable(p.order, func(a, b int) bool {
			sa, sb := p.sprints[p.order[a]], p.sprints[p.order[b]]
			if sa.last != sb.last {
				return sa.last > sb.last
			}
			return p.order[a] < p.order[b]
		})

		pr := ProjectRollup{Name: pname, Open: p.open, Blocked: p.blocked, lastUpdated: p.last}
		for i, sn := range p.order {
			if i >= sprintsN {
				pr.MoreSprints = len(p.order) - sprintsN
				break
			}
			s := p.sprints[sn]
			pr.Sprints = append(pr.Sprints, SprintRollup{Name: sn, Open: s.open, Blocked: s.blocked, lastUpdated: s.last})
		}
		rollups = append(rollups, pr)
	}

	sort.SliceStable(rollups, func(a, b int) bool {
		if rollups[a].lastUpdated != rollups[b].lastUpdated {
			return rollups[a].lastUpdated > rollups[b].lastUpdated
		}
		return rollups[a].Name < rollups[b].Name
	})

	model := OverviewModel{TotalProjects: len(rollups)}
	if len(rollups) > projectsN {
		model.MoreProjects = len(rollups) - projectsN
		rollups = rollups[:projectsN]
	}
	model.Projects = rollups
	return model
}

// Overview prints the cross-project dashboard. total is the API-reported count
// of matching items; fetched is how many were actually retrieved (page cap).
func Overview(wps []api.WorkPackage, projectsN, sprintsN, total, fetched int) {
	model := BuildOverview(wps, projectsN, sprintsN)
	if len(model.Projects) == 0 {
		fmt.Println("No open work assigned to you.")
		return
	}

	fmt.Printf("My open work — %d project(s), most recent first\n\n", model.TotalProjects)

	for _, p := range model.Projects {
		header := fmt.Sprintf("%s — %d open", p.Name, p.Open)
		if p.Blocked > 0 {
			header += fmt.Sprintf(", %d blocked", p.Blocked)
		}
		fmt.Println(header)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, s := range p.Sprints {
			blocked := ""
			if s.Blocked > 0 {
				blocked = fmt.Sprintf("%d blocked", s.Blocked)
			}
			fmt.Fprintf(w, "  %s\t%d open\t%s\n", s.Name, s.Open, blocked)
		}
		w.Flush()
		if p.MoreSprints > 0 {
			fmt.Printf("  … +%d more sprint(s)\n", p.MoreSprints)
		}
		fmt.Println()
	}

	if model.MoreProjects > 0 {
		fmt.Printf("Showing top %d of %d projects — use --projects N, or 'op my -p <project>' for detail.\n",
			projectsN, model.TotalProjects)
	}
	if total > fetched {
		fmt.Printf("Note: %d open items total; summarized the %d most recently updated.\n", total, fetched)
	}
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
