package git

import (
	"context"
	"fmt"
	"time"

	"github.com/kwo/gitterbug/pkg/tui"
)

func Run(ctx context.Context, ui *tui.TUI) {
	ui.SetStatus("finding git repositories")

	ui.SetStatus("waiting for processes to finish")

	// git status
	// git fetch origin
	// git pull current branch if status is clean

	const x = 3
	var p []*tui.ProgressLine
	for i := 0; i < x; i++ {
		p = append(p, ui.AddProcess())
	}

	tkr := time.NewTicker(time.Second * 1)
	defer tkr.Stop()

	for {
		select {
		case <-ctx.Done():
			ui.SetStatus("cancelled")
			return
		case ts := <-tkr.C:
			timeStr := ts.Format(time.TimeOnly)
			ui.SetStatus(fmt.Sprintf("ts: %s", timeStr))
			p[0].SetStatus(fmt.Sprintf("progress 0: %s", timeStr))
			p[1].SetStatus(fmt.Sprintf("progress 1: %s", timeStr))
			p[2].SetStatus(fmt.Sprintf("progress 2: %s", timeStr))
		}
	}
}
